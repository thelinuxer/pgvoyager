package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/atoulan/pgvoyager/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isValidIdentifier(s string) bool {
	return identifierRegex.MatchString(s)
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

	// Scan rows
	var data []map[string]any
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
	rows, err := pool.Query(ctx, req.SQL, req.Params...)
	duration := time.Since(start).Seconds() * 1000

	if err != nil {
		c.JSON(http.StatusOK, models.QueryResult{
			Error:    err.Error(),
			Duration: duration,
		})
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
			c.JSON(http.StatusOK, models.QueryResult{
				Error:    err.Error(),
				Duration: duration,
			})
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
