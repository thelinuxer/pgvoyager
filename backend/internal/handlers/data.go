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

func isValidIdentifier(s string) bool {
	return identifierRegex.MatchString(s)
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
			row[string(fd.Name)] = values[i]
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
		row[string(fd.Name)] = values[i]
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
	columns := make([]models.ColumnInfo, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = models.ColumnInfo{
			Name:     string(fd.Name),
			DataType: fmt.Sprintf("%d", fd.DataTypeOID),
		}
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
			row[string(fd.Name)] = values[i]
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
