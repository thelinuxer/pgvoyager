package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/thelinuxer/pgvoyager/internal/models"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// convertValue converts []byte values (e.g. XML columns) to string
// so they aren't base64-encoded in JSON responses.
func convertValue(v any) any {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}

func isValidIdentifier(s string) bool {
	return identifierRegex.MatchString(s)
}

// getTypeNames converts PostgreSQL OIDs to type names using pg_type
func getTypeNames(ctx context.Context, pool interface{ Query(context.Context, string, ...any) (pgx.Rows, error) }, oids []uint32) (map[uint32]string, error) {
	if len(oids) == 0 {
		return make(map[uint32]string), nil
	}

	// Build query with OID list
	placeholders := make([]string, len(oids))
	args := make([]any, len(oids))
	for i, oid := range oids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = oid
	}

	query := fmt.Sprintf(`
		SELECT oid, typname
		FROM pg_type
		WHERE oid IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	typeNames := make(map[uint32]string)
	for rows.Next() {
		var oid uint32
		var typname string
		if err := rows.Scan(&oid, &typname); err != nil {
			return nil, err
		}
		typeNames[oid] = typname
	}

	return typeNames, nil
}

// ColumnFKInfo holds FK information for a column identified by table OID and attribute number
type ColumnFKInfo struct {
	IsPrimaryKey bool
	IsForeignKey bool
	FKReference  *models.FKRef
}

// getColumnFKInfo looks up primary key and foreign key information for columns based on their table OID and attribute number
func getColumnFKInfo(ctx context.Context, pool interface{ Query(context.Context, string, ...any) (pgx.Rows, error) }, tableOIDs []uint32) (map[uint32]map[uint16]ColumnFKInfo, error) {
	if len(tableOIDs) == 0 {
		return make(map[uint32]map[uint16]ColumnFKInfo), nil
	}

	// Build query with table OID list
	placeholders := make([]string, len(tableOIDs))
	args := make([]any, len(tableOIDs))
	for i, oid := range tableOIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = oid
	}

	// Query to get PK and FK info for columns in the specified tables
	query := fmt.Sprintf(`
		WITH pk_cols AS (
			SELECT
				c.conrelid as table_oid,
				unnest(c.conkey) as attnum
			FROM pg_constraint c
			WHERE c.contype = 'p'
			AND c.conrelid IN (%s)
		),
		fk_cols AS (
			SELECT
				c.conrelid as table_oid,
				u.key_attnum as attnum,
				ref_ns.nspname as ref_schema,
				ref_cls.relname as ref_table,
				ref_att.attname as ref_column
			FROM pg_constraint c
			JOIN pg_class ref_cls ON ref_cls.oid = c.confrelid
			JOIN pg_namespace ref_ns ON ref_ns.oid = ref_cls.relnamespace
			CROSS JOIN LATERAL unnest(c.conkey, c.confkey) WITH ORDINALITY AS u(key_attnum, ref_attnum, ord)
			JOIN pg_attribute ref_att ON ref_att.attrelid = c.confrelid AND ref_att.attnum = u.ref_attnum
			WHERE c.contype = 'f'
			AND c.conrelid IN (%s)
		)
		SELECT
			a.attrelid as table_oid,
			a.attnum,
			COALESCE(pk.attnum IS NOT NULL, false) as is_pk,
			COALESCE(fk.attnum IS NOT NULL, false) as is_fk,
			fk.ref_schema,
			fk.ref_table,
			fk.ref_column
		FROM pg_attribute a
		LEFT JOIN pk_cols pk ON pk.table_oid = a.attrelid AND pk.attnum = a.attnum
		LEFT JOIN fk_cols fk ON fk.table_oid = a.attrelid AND fk.attnum = a.attnum
		WHERE a.attrelid IN (%s)
		AND a.attnum > 0
		AND NOT a.attisdropped
		AND (pk.attnum IS NOT NULL OR fk.attnum IS NOT NULL)
	`, strings.Join(placeholders, ", "), strings.Join(placeholders, ", "), strings.Join(placeholders, ", "))

	// PostgreSQL reuses the same parameter for all occurrences of $1, $2, etc.
	// so we only need to pass args once
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint32]map[uint16]ColumnFKInfo)
	for rows.Next() {
		var tableOID uint32
		var attnum uint16
		var isPK, isFK bool
		var refSchema, refTable, refColumn *string

		if err := rows.Scan(&tableOID, &attnum, &isPK, &isFK, &refSchema, &refTable, &refColumn); err != nil {
			return nil, err
		}

		if result[tableOID] == nil {
			result[tableOID] = make(map[uint16]ColumnFKInfo)
		}

		info := ColumnFKInfo{
			IsPrimaryKey: isPK,
			IsForeignKey: isFK,
		}
		if isFK && refSchema != nil && refTable != nil && refColumn != nil {
			info.FKReference = &models.FKRef{
				Schema: *refSchema,
				Table:  *refTable,
				Column: *refColumn,
			}
		}
		result[tableOID][attnum] = info
	}

	return result, nil
}

// buildErrorResult creates a QueryResult with detailed error information from PgError
// positionOffset is added to the error position (for multi-statement queries)
func buildErrorResult(err error, duration float64, positionOffset int) models.QueryResult {
	result := models.QueryResult{
		Error:    err.Error(),
		Duration: duration,
	}

	// Try to extract PostgreSQL-specific error details
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		result.Error = pgErr.Message
		if pgErr.Code != "" {
			result.Error += " (SQLSTATE " + pgErr.Code + ")"
		}
		if pgErr.Position > 0 {
			// Add offset for multi-statement queries
			result.ErrorPosition = int(pgErr.Position) + positionOffset
		}
		if pgErr.Hint != "" {
			result.ErrorHint = pgErr.Hint
		}
		if pgErr.Detail != "" {
			result.ErrorDetail = pgErr.Detail
		}
	}

	return result
}

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func GetTableData(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	// Validate identifiers
	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema or table name"})
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))
	orderBy := c.Query("orderBy")
	orderDir := c.DefaultQuery("orderDir", "ASC")
	filterColumn := c.Query("filterColumn")
	filterValue := c.Query("filterValue")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 100
	}
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "ASC"
	}

	// Validate filter column if provided
	hasFilter := filterColumn != "" && filterValue != ""
	if hasFilter && !isValidIdentifier(filterColumn) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filter column name"})
		return
	}

	// Get column info with FK references
	columns, err := getTableColumnInfo(ctx, pool, schema, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build WHERE clause for filter
	var whereClause string
	var queryArgs []any
	if hasFilter {
		whereClause = fmt.Sprintf(" WHERE %s = $1", quoteIdentifier(filterColumn))
		queryArgs = append(queryArgs, filterValue)
	}

	// Get total row count (with filter if applicable)
	var totalRows int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s%s", quoteIdentifier(schema), quoteIdentifier(table), whereClause)
	if err := pool.QueryRow(ctx, countQuery, queryArgs...).Scan(&totalRows); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build data query
	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf("SELECT * FROM %s.%s%s", quoteIdentifier(schema), quoteIdentifier(table), whereClause)

	if orderBy != "" && isValidIdentifier(orderBy) {
		dataQuery += fmt.Sprintf(" ORDER BY %s %s", quoteIdentifier(orderBy), orderDir)
	}

	dataQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Get field descriptions
	fieldDescs := rows.FieldDescriptions()

	// Scan rows - initialize to empty slice to avoid null in JSON
	data := []map[string]any{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		row := make(map[string]any)
		for i, fd := range fieldDescs {
			row[string(fd.Name)] = convertValue(values[i])
		}
		data = append(data, row)
	}

	totalPages := int(totalRows) / pageSize
	if int(totalRows)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, models.TableDataResponse{
		Columns:    columns,
		Rows:       data,
		TotalRows:  totalRows,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func getTableColumnInfo(ctx context.Context, pool interface{ Query(context.Context, string, ...any) (pgx.Rows, error) }, schema, table string) ([]models.ColumnInfo, error) {
	query := `
		SELECT
			a.attname as name,
			pg_catalog.format_type(a.atttypid, a.atttypmod) as data_type,
			COALESCE(pk.is_pk, false) as is_primary_key,
			COALESCE(fk.is_fk, false) as is_foreign_key,
			fk.ref_schema,
			fk.ref_table,
			fk.ref_column
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN LATERAL (
			SELECT true as is_pk
			FROM pg_constraint con
			WHERE con.conrelid = c.oid
			  AND con.contype = 'p'
			  AND a.attnum = ANY(con.conkey)
		) pk ON true
		LEFT JOIN LATERAL (
			SELECT
				true as is_fk,
				nf.nspname as ref_schema,
				cf.relname as ref_table,
				af.attname as ref_column
			FROM pg_constraint con
			JOIN pg_class cf ON cf.oid = con.confrelid
			JOIN pg_namespace nf ON nf.oid = cf.relnamespace
			JOIN pg_attribute af ON af.attrelid = con.confrelid
				AND af.attnum = con.confkey[array_position(con.conkey, a.attnum)]
			WHERE con.conrelid = c.oid
			  AND con.contype = 'f'
			  AND a.attnum = ANY(con.conkey)
			LIMIT 1
		) fk ON true
		WHERE n.nspname = $1
		  AND c.relname = $2
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY a.attnum
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var refSchema, refTable, refColumn *string

		if err := rows.Scan(
			&col.Name, &col.DataType, &col.IsPrimaryKey, &col.IsForeignKey,
			&refSchema, &refTable, &refColumn,
		); err != nil {
			return nil, err
		}

		if col.IsForeignKey && refSchema != nil {
			col.FKReference = &models.FKRef{
				Schema: *refSchema,
				Table:  *refTable,
				Column: *refColumn,
			}
		}

		columns = append(columns, col)
	}

	return columns, nil
}

func GetTableRowCount(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema or table name"})
		return
	}

	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", quoteIdentifier(schema), quoteIdentifier(table))
	if err := pool.QueryRow(ctx, query).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func GetForeignKeyPreview(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")
	column := c.Param("column")
	value := c.Param("value")

	if !isValidIdentifier(schema) || !isValidIdentifier(table) || !isValidIdentifier(column) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid identifier"})
		return
	}

	// Get column info with FK references
	columns, err := getTableColumnInfo(ctx, pool, schema, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the row
	query := fmt.Sprintf("SELECT * FROM %s.%s WHERE %s = $1 LIMIT 1",
		quoteIdentifier(schema), quoteIdentifier(table), quoteIdentifier(column))

	rows, err := pool.Query(ctx, query, value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	values, err := rows.Values()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fieldDescs := rows.FieldDescriptions()
	row := make(map[string]any)
	for i, fd := range fieldDescs {
		row[string(fd.Name)] = convertValue(values[i])
	}

	c.JSON(http.StatusOK, models.ForeignKeyPreview{
		Schema:  schema,
		Table:   table,
		Columns: columns,
		Row:     row,
	})
}

// StatementInfo holds a SQL statement and its position in the original query
type StatementInfo struct {
	SQL    string
	Offset int // 0-based byte offset in original SQL where this statement starts
}

// splitStatements splits SQL into individual statements, handling string literals
func splitStatements(sql string) []StatementInfo {
	var statements []StatementInfo
	var current strings.Builder
	inString := false
	stringChar := rune(0)
	stmtStartByte := 0

	byteOffset := 0
	for i, ch := range sql {
		current.WriteRune(ch)
		charLen := len(string(ch))

		if !inString {
			if ch == '\'' || ch == '"' {
				inString = true
				stringChar = ch
			} else if ch == ';' {
				stmt := strings.TrimSpace(current.String())
				// Remove trailing semicolon for cleaner statement
				stmt = strings.TrimSuffix(stmt, ";")
				stmt = strings.TrimSpace(stmt)
				if len(stmt) > 0 {
					// Find actual start by skipping whitespace from stmtStartByte
					actualStart := stmtStartByte
					for actualStart < len(sql) && (sql[actualStart] == ' ' || sql[actualStart] == '\t' || sql[actualStart] == '\n' || sql[actualStart] == '\r') {
						actualStart++
					}
					statements = append(statements, StatementInfo{SQL: stmt, Offset: actualStart})
				}
				current.Reset()
				stmtStartByte = byteOffset + charLen
			}
		} else {
			if ch == stringChar {
				// Check for escaped quote (two consecutive quotes)
				if i+1 < len(sql) && rune(sql[i+1]) == stringChar {
					byteOffset += charLen
					continue
				}
				inString = false
			}
		}
		byteOffset += charLen
	}

	// Handle last statement (may not end with semicolon)
	stmt := strings.TrimSpace(current.String())
	if len(stmt) > 0 {
		// Find actual start by skipping whitespace from stmtStartByte
		actualStart := stmtStartByte
		for actualStart < len(sql) && (sql[actualStart] == ' ' || sql[actualStart] == '\t' || sql[actualStart] == '\n' || sql[actualStart] == '\r') {
			actualStart++
		}
		statements = append(statements, StatementInfo{SQL: stmt, Offset: actualStart})
	}

	return statements
}

// isSelectStatement checks if a statement is a SELECT (returns rows)
func isSelectStatement(sql string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	return strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "TABLE") ||
		strings.HasPrefix(upper, "VALUES")
}

func ExecuteQuery(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var req models.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start := time.Now()

	// Split into statements and handle multi-statement queries
	statements := splitStatements(req.SQL)

	// Track the offset of the statement being executed (for error position)
	currentOffset := 0

	// If multiple statements, execute non-SELECT statements first with Exec
	// then execute the final SELECT with Query
	if len(statements) > 1 && len(req.Params) == 0 {
		var selectStmtInfo *StatementInfo
		for i := range statements {
			stmtInfo := &statements[i]
			if isSelectStatement(stmtInfo.SQL) {
				selectStmtInfo = stmtInfo
			} else {
				// Execute non-SELECT statements (SET, CREATE, etc.)
				_, err := pool.Exec(ctx, stmtInfo.SQL)
				if err != nil {
					duration := time.Since(start).Seconds() * 1000
					c.JSON(http.StatusOK, buildErrorResult(err, duration, stmtInfo.Offset))
					return
				}
			}
		}

		// If there was a SELECT statement, execute it
		if selectStmtInfo != nil {
			req.SQL = selectStmtInfo.SQL
			currentOffset = selectStmtInfo.Offset
		} else {
			// All statements were non-SELECT, return success
			duration := time.Since(start).Seconds() * 1000
			c.JSON(http.StatusOK, models.QueryResult{
				Columns:  []models.ColumnInfo{},
				Rows:     []map[string]any{},
				RowCount: 0,
				Duration: duration,
			})
			return
		}
	} else if len(statements) == 1 {
		// Single statement - use its offset (usually 0, but could have leading whitespace)
		currentOffset = statements[0].Offset
	}

	rows, err := pool.Query(ctx, req.SQL, req.Params...)
	duration := time.Since(start).Seconds() * 1000

	if err != nil {
		c.JSON(http.StatusOK, buildErrorResult(err, duration, currentOffset))
		return
	}
	defer rows.Close()

	fieldDescs := rows.FieldDescriptions()

	// Collect unique type OIDs and table OIDs
	typeOIDSet := make(map[uint32]bool)
	tableOIDSet := make(map[uint32]bool)
	for _, fd := range fieldDescs {
		typeOIDSet[fd.DataTypeOID] = true
		if fd.TableOID != 0 {
			tableOIDSet[fd.TableOID] = true
		}
	}

	typeOIDs := make([]uint32, 0, len(typeOIDSet))
	for oid := range typeOIDSet {
		typeOIDs = append(typeOIDs, oid)
	}

	tableOIDs := make([]uint32, 0, len(tableOIDSet))
	for oid := range tableOIDSet {
		tableOIDs = append(tableOIDs, oid)
	}

	// Look up type names
	typeNames, err := getTypeNames(ctx, pool, typeOIDs)
	if err != nil {
		// Fall back to OID if type lookup fails
		typeNames = make(map[uint32]string)
	}

	// Look up FK info for columns that come from real tables
	fkInfo, err := getColumnFKInfo(ctx, pool, tableOIDs)
	if err != nil {
		// Continue without FK info if lookup fails
		fkInfo = make(map[uint32]map[uint16]ColumnFKInfo)
	}

	columns := make([]models.ColumnInfo, len(fieldDescs))
	for i, fd := range fieldDescs {
		typeName := typeNames[fd.DataTypeOID]
		if typeName == "" {
			typeName = fmt.Sprintf("oid:%d", fd.DataTypeOID)
		}
		col := models.ColumnInfo{
			Name:     string(fd.Name),
			DataType: typeName,
		}

		// Add FK info if available for this column
		if fd.TableOID != 0 {
			if tableInfo, ok := fkInfo[fd.TableOID]; ok {
				if colInfo, ok := tableInfo[fd.TableAttributeNumber]; ok {
					col.IsPrimaryKey = colInfo.IsPrimaryKey
					col.IsForeignKey = colInfo.IsForeignKey
					col.FKReference = colInfo.FKReference
				}
			}
		}

		columns[i] = col
	}

	var data []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			c.JSON(http.StatusOK, buildErrorResult(err, duration, currentOffset))
			return
		}

		row := make(map[string]any)
		for i, fd := range fieldDescs {
			row[string(fd.Name)] = convertValue(values[i])
		}
		data = append(data, row)
	}

	c.JSON(http.StatusOK, models.QueryResult{
		Columns:  columns,
		Rows:     data,
		RowCount: len(data),
		Duration: duration,
	})
}

func ExplainQuery(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var req models.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	explainQuery := "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + req.SQL

	start := time.Now()
	rows, err := pool.Query(ctx, explainQuery, req.Params...)
	duration := time.Since(start).Seconds() * 1000

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		planLines = append(planLines, line)
	}

	c.JSON(http.StatusOK, models.ExplainResult{
		Plan:     strings.Join(planLines, "\n"),
		Duration: duration,
	})
}

func InsertRow(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema or table name"})
		return
	}

	var req models.InsertRowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No data provided"})
		return
	}

	// Build INSERT query
	columns := make([]string, 0, len(req.Data))
	placeholders := make([]string, 0, len(req.Data))
	values := make([]any, 0, len(req.Data))
	i := 1

	for col, val := range req.Data {
		if !isValidIdentifier(col) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid column name: %s", col)})
			return
		}
		columns = append(columns, quoteIdentifier(col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s.%s (%s) VALUES (%s) RETURNING *",
		quoteIdentifier(schema),
		quoteIdentifier(table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	rows, err := pool.Query(ctx, query, values...)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert succeeded but no row returned"})
		return
	}

	rowValues, err := rows.Values()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fieldDescs := rows.FieldDescriptions()
	insertedRow := make(map[string]any)
	for i, fd := range fieldDescs {
		insertedRow[string(fd.Name)] = rowValues[i]
	}

	c.JSON(http.StatusCreated, models.CrudResponse{
		Success:      true,
		RowsAffected: 1,
		Message:      "Row inserted successfully",
		InsertedRow:  insertedRow,
	})
}

func UpdateRow(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema or table name"})
		return
	}

	var req models.UpdateRowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.PrimaryKey) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Primary key required"})
		return
	}

	if len(req.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No data to update"})
		return
	}

	// Build SET clause
	setClauses := make([]string, 0, len(req.Data))
	values := make([]any, 0)
	paramNum := 1

	for col, val := range req.Data {
		if !isValidIdentifier(col) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid column name: %s", col)})
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), paramNum))
		values = append(values, val)
		paramNum++
	}

	// Build WHERE clause from primary key
	whereClauses := make([]string, 0, len(req.PrimaryKey))
	for col, val := range req.PrimaryKey {
		if !isValidIdentifier(col) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid primary key column: %s", col)})
			return
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), paramNum))
		values = append(values, val)
		paramNum++
	}

	query := fmt.Sprintf(
		"UPDATE %s.%s SET %s WHERE %s",
		quoteIdentifier(schema),
		quoteIdentifier(table),
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	result, err := pool.Exec(ctx, query, values...)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No row found with the specified primary key"})
		return
	}

	c.JSON(http.StatusOK, models.CrudResponse{
		Success:      true,
		RowsAffected: rowsAffected,
		Message:      "Row updated successfully",
	})
}

func DeleteRow(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema or table name"})
		return
	}

	var req models.DeleteRowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.PrimaryKey) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Primary key required"})
		return
	}

	// Build WHERE clause from primary key
	whereClauses := make([]string, 0, len(req.PrimaryKey))
	values := make([]any, 0)
	paramNum := 1

	for col, val := range req.PrimaryKey {
		if !isValidIdentifier(col) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid primary key column: %s", col)})
			return
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), paramNum))
		values = append(values, val)
		paramNum++
	}

	query := fmt.Sprintf(
		"DELETE FROM %s.%s WHERE %s",
		quoteIdentifier(schema),
		quoteIdentifier(table),
		strings.Join(whereClauses, " AND "),
	)

	result, err := pool.Exec(ctx, query, values...)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No row found with the specified primary key"})
		return
	}

	c.JSON(http.StatusOK, models.CrudResponse{
		Success:      true,
		RowsAffected: rowsAffected,
		Message:      "Row deleted successfully",
	})
}

func DropTable(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema or table name"})
		return
	}

	// Optional: check for CASCADE option
	var req struct {
		Cascade bool `json:"cascade"`
	}
	c.ShouldBindJSON(&req)

	query := fmt.Sprintf("DROP TABLE %s.%s", quoteIdentifier(schema), quoteIdentifier(table))
	if req.Cascade {
		query += " CASCADE"
	}

	_, err := pool.Exec(ctx, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Table %s.%s dropped successfully", schema, table),
	})
}

func CreateSchema(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if !isValidIdentifier(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema name"})
		return
	}

	query := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(req.Name))
	_, err := pool.Exec(ctx, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Schema %s created successfully", req.Name),
	})
}

func DropSchema(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	if !isValidIdentifier(schema) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema name"})
		return
	}

	var req struct {
		Cascade bool `json:"cascade"`
	}
	c.ShouldBindJSON(&req)

	query := fmt.Sprintf("DROP SCHEMA %s", quoteIdentifier(schema))
	if req.Cascade {
		query += " CASCADE"
	}

	_, err := pool.Exec(ctx, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Schema %s dropped successfully", schema),
	})
}

func CreateTable(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	if !isValidIdentifier(schema) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema name"})
		return
	}

	type ColumnDef struct {
		Name       string  `json:"name"`
		Type       string  `json:"type"`
		Nullable   bool    `json:"nullable"`
		Default    *string `json:"default"`
		PrimaryKey bool    `json:"primaryKey"`
	}

	var req struct {
		Name    string      `json:"name"`
		Columns []ColumnDef `json:"columns"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if !isValidIdentifier(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table name"})
		return
	}

	if len(req.Columns) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one column is required"})
		return
	}

	var colDefs []string
	var pkCols []string

	for _, col := range req.Columns {
		if !isValidIdentifier(col.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid column name: %s", col.Name)})
			return
		}

		def := fmt.Sprintf("%s %s", quoteIdentifier(col.Name), col.Type)
		if !col.Nullable {
			def += " NOT NULL"
		}
		if col.Default != nil && *col.Default != "" {
			def += " DEFAULT " + *col.Default
		}
		colDefs = append(colDefs, def)

		if col.PrimaryKey {
			pkCols = append(pkCols, quoteIdentifier(col.Name))
		}
	}

	if len(pkCols) > 0 {
		colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}

	query := fmt.Sprintf("CREATE TABLE %s.%s (\n  %s\n)",
		quoteIdentifier(schema),
		quoteIdentifier(req.Name),
		strings.Join(colDefs, ",\n  "))

	_, err := pool.Exec(ctx, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Table %s.%s created successfully", schema, req.Name),
	})
}

func AddConstraint(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema or table name"})
		return
	}

	var req struct {
		Type       string   `json:"type"`
		Name       string   `json:"name"`
		Columns    []string `json:"columns"`
		RefSchema  string   `json:"refSchema"`
		RefTable   string   `json:"refTable"`
		RefColumns []string `json:"refColumns"`
		OnDelete   string   `json:"onDelete"`
		OnUpdate   string   `json:"onUpdate"`
		Expression string   `json:"expression"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate columns
	for _, col := range req.Columns {
		if !isValidIdentifier(col) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid column name: %s", col)})
			return
		}
	}

	// Auto-generate constraint name if not provided
	constraintName := req.Name
	if constraintName == "" {
		constraintName = fmt.Sprintf("%s_%s_%s", table, req.Type, strings.Join(req.Columns, "_"))
	}
	if !isValidIdentifier(constraintName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid constraint name"})
		return
	}

	var ddl string
	qualifiedTable := fmt.Sprintf("%s.%s", quoteIdentifier(schema), quoteIdentifier(table))

	switch strings.ToLower(req.Type) {
	case "fk":
		if len(req.Columns) == 0 || req.RefTable == "" || len(req.RefColumns) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "FK constraint requires columns, refTable, and refColumns"})
			return
		}
		if !isValidIdentifier(req.RefTable) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reference table name"})
			return
		}
		for _, col := range req.RefColumns {
			if !isValidIdentifier(col) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid reference column name: %s", col)})
				return
			}
		}

		refSchemaPrefix := ""
		if req.RefSchema != "" {
			if !isValidIdentifier(req.RefSchema) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reference schema name"})
				return
			}
			refSchemaPrefix = quoteIdentifier(req.RefSchema) + "."
		}

		quotedCols := make([]string, len(req.Columns))
		for i, col := range req.Columns {
			quotedCols[i] = quoteIdentifier(col)
		}
		quotedRefCols := make([]string, len(req.RefColumns))
		for i, col := range req.RefColumns {
			quotedRefCols[i] = quoteIdentifier(col)
		}

		ddl = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s%s(%s)",
			qualifiedTable,
			quoteIdentifier(constraintName),
			strings.Join(quotedCols, ", "),
			refSchemaPrefix,
			quoteIdentifier(req.RefTable),
			strings.Join(quotedRefCols, ", "))

		if req.OnDelete != "" {
			ddl += " ON DELETE " + req.OnDelete
		}
		if req.OnUpdate != "" {
			ddl += " ON UPDATE " + req.OnUpdate
		}

	case "unique":
		if len(req.Columns) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "UNIQUE constraint requires at least one column"})
			return
		}

		quotedCols := make([]string, len(req.Columns))
		for i, col := range req.Columns {
			quotedCols[i] = quoteIdentifier(col)
		}

		ddl = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s)",
			qualifiedTable,
			quoteIdentifier(constraintName),
			strings.Join(quotedCols, ", "))

	case "check":
		if req.Expression == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "CHECK constraint requires an expression"})
			return
		}

		ddl = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)",
			qualifiedTable,
			quoteIdentifier(constraintName),
			req.Expression)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid constraint type. Must be 'fk', 'unique', or 'check'"})
		return
	}

	_, err := pool.Exec(ctx, ddl)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Constraint %s added to %s.%s", constraintName, schema, table),
	})
}
