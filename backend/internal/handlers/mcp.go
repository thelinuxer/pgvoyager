package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thelinuxer/pgvoyager/internal/claude"
	"github.com/thelinuxer/pgvoyager/internal/database"
)

// getMCPPool gets the database pool for the current Claude session
func getMCPPool(c *gin.Context) (*database.ConnectionManager, string, bool) {
	sessionID := c.GetHeader("X-Claude-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Claude-Session-ID header"})
		return nil, "", false
	}

	// Get session to find the connection ID
	claudeManager := claude.GetManager()
	session, ok := claudeManager.GetSession(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Claude session not found"})
		return nil, "", false
	}

	// Get the database manager and check connection
	dbManager := database.GetManager()
	if !dbManager.IsConnected(session.ConnectionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Database not connected"})
		return nil, "", false
	}

	return dbManager, session.ConnectionID, true
}

// MCPGetConnectionInfo returns info about the current connection
func MCPGetConnectionInfo(c *gin.Context) {
	sessionID := c.GetHeader("X-Claude-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Claude-Session-ID header"})
		return
	}

	claudeManager := claude.GetManager()
	session, ok := claudeManager.GetSession(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Claude session not found"})
		return
	}

	dbManager := database.GetManager()
	conn, err := dbManager.Get(session.ConnectionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connection not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"database":     conn.Database,
		"host":         conn.Host,
		"port":         conn.Port,
		"user":         conn.Username,
		"is_connected": dbManager.IsConnected(session.ConnectionID),
	})
}

// MCPListSchemas lists all schemas
func MCPListSchemas(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT
			n.nspname as name,
			pg_catalog.pg_get_userbyid(n.nspowner) as owner,
			(SELECT count(*) FROM pg_catalog.pg_class c
			 WHERE c.relnamespace = n.oid AND c.relkind = 'r') as table_count
		FROM pg_catalog.pg_namespace n
		WHERE n.nspname NOT LIKE 'pg_%'
		  AND n.nspname != 'information_schema'
		ORDER BY n.nspname
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var schemas []map[string]interface{}
	for rows.Next() {
		var name, owner string
		var tableCount int64
		if err := rows.Scan(&name, &owner, &tableCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		schemas = append(schemas, map[string]interface{}{
			"name":        name,
			"owner":       owner,
			"table_count": tableCount,
		})
	}

	// Return as formatted JSON for Claude
	result, _ := json.MarshalIndent(schemas, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPListTables lists tables
func MCPListTables(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaFilter := c.Query("schema")

	query := `
		SELECT
			n.nspname as schema,
			c.relname as name,
			pg_catalog.pg_get_userbyid(c.relowner) as owner,
			c.reltuples::bigint as row_count,
			pg_catalog.pg_size_pretty(pg_catalog.pg_table_size(c.oid)) as size,
			COALESCE(obj_description(c.oid), '') as comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r'
		  AND n.nspname NOT LIKE 'pg_%'
		  AND n.nspname != 'information_schema'
	`

	args := []interface{}{}
	if schemaFilter != "" {
		query += " AND n.nspname = $1"
		args = append(args, schemaFilter)
	}
	query += " ORDER BY n.nspname, c.relname"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var schema, name, owner, size, comment string
		var rowCount int64
		if err := rows.Scan(&schema, &name, &owner, &rowCount, &size, &comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tables = append(tables, map[string]interface{}{
			"schema":    schema,
			"name":      name,
			"owner":     owner,
			"row_count": rowCount,
			"size":      size,
			"comment":   comment,
		})
	}

	result, _ := json.MarshalIndent(tables, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPGetTableInfo gets table details
func MCPGetTableInfo(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	query := `
		SELECT
			n.nspname as schema,
			c.relname as name,
			pg_catalog.pg_get_userbyid(c.relowner) as owner,
			c.reltuples::bigint as row_count,
			pg_catalog.pg_size_pretty(pg_catalog.pg_table_size(c.oid)) as size,
			pg_catalog.pg_size_pretty(pg_catalog.pg_indexes_size(c.oid)) as indexes_size,
			pg_catalog.pg_size_pretty(pg_catalog.pg_total_relation_size(c.oid)) as total_size,
			EXISTS(SELECT 1 FROM pg_constraint con WHERE con.conrelid = c.oid AND con.contype = 'p') as has_pk,
			COALESCE(obj_description(c.oid), '') as comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r'
		  AND n.nspname = $1
		  AND c.relname = $2
	`

	var schemaName, tableName, owner, size, indexesSize, totalSize, comment string
	var rowCount int64
	var hasPK bool

	err := pool.QueryRow(ctx, query, schema, table).Scan(
		&schemaName, &tableName, &owner, &rowCount, &size, &indexesSize, &totalSize, &hasPK, &comment,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	info := map[string]interface{}{
		"schema":       schemaName,
		"name":         tableName,
		"owner":        owner,
		"row_count":    rowCount,
		"size":         size,
		"indexes_size": indexesSize,
		"total_size":   totalSize,
		"has_pk":       hasPK,
		"comment":      comment,
	}

	result, _ := json.MarshalIndent(info, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPGetColumns gets column info for a table
func MCPGetColumns(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	query := `
		SELECT
			a.attname as name,
			a.attnum as position,
			pg_catalog.format_type(a.atttypid, a.atttypmod) as data_type,
			NOT a.attnotnull as is_nullable,
			pg_catalog.pg_get_expr(d.adbin, d.adrelid) as default_value,
			COALESCE(pk.is_pk, false) as is_primary_key,
			COALESCE(fk.is_fk, false) as is_foreign_key,
			fk.ref_schema,
			fk.ref_table,
			fk.ref_column,
			COALESCE(col_description(c.oid, a.attnum), '') as comment
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var name, dataType, comment string
		var position int
		var isNullable, isPrimaryKey, isForeignKey bool
		var defaultValue, refSchema, refTable, refColumn *string

		if err := rows.Scan(&name, &position, &dataType, &isNullable, &defaultValue,
			&isPrimaryKey, &isForeignKey, &refSchema, &refTable, &refColumn, &comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		col := map[string]interface{}{
			"name":           name,
			"position":       position,
			"data_type":      dataType,
			"is_nullable":    isNullable,
			"is_primary_key": isPrimaryKey,
			"is_foreign_key": isForeignKey,
			"comment":        comment,
		}

		if defaultValue != nil {
			col["default_value"] = *defaultValue
		}
		if isForeignKey && refSchema != nil {
			col["fk_reference"] = map[string]string{
				"schema": *refSchema,
				"table":  *refTable,
				"column": *refColumn,
			}
		}

		columns = append(columns, col)
	}

	result, _ := json.MarshalIndent(columns, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPExecuteQuery executes a SQL query
func MCPExecuteQuery(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	var req struct {
		SQL   string `json:"sql" binding:"required"`
		Limit int    `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, req.SQL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = string(fd.Name)
	}

	// Fetch rows
	var results []map[string]interface{}
	count := 0
	for rows.Next() && count < req.Limit {
		values, err := rows.Values()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
		count++
	}

	output := map[string]interface{}{
		"columns":   columns,
		"rows":      results,
		"row_count": count,
	}

	result, _ := json.MarshalIndent(output, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPListViews lists views
func MCPListViews(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaFilter := c.Query("schema")

	query := `
		SELECT
			n.nspname as schema,
			c.relname as name,
			pg_catalog.pg_get_userbyid(c.relowner) as owner,
			pg_get_viewdef(c.oid, true) as definition,
			COALESCE(obj_description(c.oid), '') as comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'v'
		  AND n.nspname NOT LIKE 'pg_%'
		  AND n.nspname != 'information_schema'
	`

	args := []interface{}{}
	if schemaFilter != "" {
		query += " AND n.nspname = $1"
		args = append(args, schemaFilter)
	}
	query += " ORDER BY n.nspname, c.relname"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var views []map[string]interface{}
	for rows.Next() {
		var schema, name, owner, definition, comment string
		if err := rows.Scan(&schema, &name, &owner, &definition, &comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		views = append(views, map[string]interface{}{
			"schema":     schema,
			"name":       name,
			"owner":      owner,
			"definition": definition,
			"comment":    comment,
		})
	}

	result, _ := json.MarshalIndent(views, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPListFunctions lists functions
func MCPListFunctions(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaFilter := c.Query("schema")

	query := `
		SELECT
			n.nspname as schema,
			p.proname as name,
			pg_catalog.pg_get_userbyid(p.proowner) as owner,
			pg_catalog.pg_get_function_result(p.oid) as return_type,
			pg_catalog.pg_get_function_arguments(p.oid) as arguments,
			l.lanname as language,
			COALESCE(obj_description(p.oid, 'pg_proc'), '') as comment
		FROM pg_catalog.pg_proc p
		JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		JOIN pg_catalog.pg_language l ON l.oid = p.prolang
		WHERE n.nspname NOT LIKE 'pg_%'
		  AND n.nspname != 'information_schema'
		  AND p.prokind != 'a'
	`

	args := []interface{}{}
	if schemaFilter != "" {
		query += " AND n.nspname = $1"
		args = append(args, schemaFilter)
	}
	query += " ORDER BY n.nspname, p.proname LIMIT 100"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var functions []map[string]interface{}
	for rows.Next() {
		var schema, name, owner, returnType, arguments, language, comment string
		if err := rows.Scan(&schema, &name, &owner, &returnType, &arguments, &language, &comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		functions = append(functions, map[string]interface{}{
			"schema":      schema,
			"name":        name,
			"owner":       owner,
			"return_type": returnType,
			"arguments":   arguments,
			"language":    language,
			"comment":     comment,
		})
	}

	result, _ := json.MarshalIndent(functions, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPGetForeignKeys gets foreign keys for a table
func MCPGetForeignKeys(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	query := `
		SELECT
			con.conname as name,
			array_agg(a.attname ORDER BY array_position(con.conkey, a.attnum)) as columns,
			nf.nspname as ref_schema,
			cf.relname as ref_table,
			array_agg(af.attname ORDER BY array_position(con.confkey, af.attnum)) as ref_columns,
			CASE con.confupdtype
				WHEN 'a' THEN 'NO ACTION'
				WHEN 'r' THEN 'RESTRICT'
				WHEN 'c' THEN 'CASCADE'
				WHEN 'n' THEN 'SET NULL'
				WHEN 'd' THEN 'SET DEFAULT'
			END as on_update,
			CASE con.confdeltype
				WHEN 'a' THEN 'NO ACTION'
				WHEN 'r' THEN 'RESTRICT'
				WHEN 'c' THEN 'CASCADE'
				WHEN 'n' THEN 'SET NULL'
				WHEN 'd' THEN 'SET DEFAULT'
			END as on_delete
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_class cf ON cf.oid = con.confrelid
		JOIN pg_namespace nf ON nf.oid = cf.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(con.conkey)
		JOIN pg_attribute af ON af.attrelid = cf.oid AND af.attnum = ANY(con.confkey)
		WHERE con.contype = 'f'
		  AND n.nspname = $1
		  AND c.relname = $2
		GROUP BY con.oid, con.conname, nf.nspname, cf.relname, con.confupdtype, con.confdeltype
		ORDER BY con.conname
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var fks []map[string]interface{}
	for rows.Next() {
		var name, refSchema, refTable, onUpdate, onDelete string
		var columns, refColumns []string
		if err := rows.Scan(&name, &columns, &refSchema, &refTable, &refColumns, &onUpdate, &onDelete); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fks = append(fks, map[string]interface{}{
			"name":        name,
			"columns":     columns,
			"ref_schema":  refSchema,
			"ref_table":   refTable,
			"ref_columns": refColumns,
			"on_update":   onUpdate,
			"on_delete":   onDelete,
		})
	}

	result, _ := json.MarshalIndent(fks, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}

// MCPGetEditorContent gets the current editor content
func MCPGetEditorContent(c *gin.Context) {
	sessionID := c.GetHeader("X-Claude-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Claude-Session-ID header"})
		return
	}

	claudeManager := claude.GetManager()
	state, err := claudeManager.GetEditorState(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if state == nil || state.Content == "" {
		c.JSON(http.StatusOK, gin.H{
			"content": "",
			"message": "Editor is empty or not open",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": state.Content,
	})
}

// MCPInsertToEditor inserts text into the editor
func MCPInsertToEditor(c *gin.Context) {
	sessionID := c.GetHeader("X-Claude-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Claude-Session-ID header"})
		return
	}

	var req struct {
		Text     string `json:"text" binding:"required"`
		Position *struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"position"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action := &claude.EditorActionData{
		Action: "insert",
		Text:   req.Text,
	}

	if req.Position != nil {
		action.Position = &claude.Position{
			Line:   req.Position.Line,
			Column: req.Position.Column,
		}
	}

	claudeManager := claude.GetManager()
	if err := claudeManager.SendEditorAction(sessionID, action); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MCPReplaceEditorContent replaces the entire editor content
func MCPReplaceEditorContent(c *gin.Context) {
	sessionID := c.GetHeader("X-Claude-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Claude-Session-ID header"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action := &claude.EditorActionData{
		Action: "replace",
		Text:   req.Content,
	}

	claudeManager := claude.GetManager()
	if err := claudeManager.SendEditorAction(sessionID, action); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MCPGetIndexes gets indexes for a table
func MCPGetIndexes(c *gin.Context) {
	manager, connId, ok := getMCPPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")
	table := c.Param("table")

	query := `
		SELECT
			i.relname as name,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) as columns,
			ix.indisunique as is_unique,
			ix.indisprimary as is_primary,
			am.amname as type,
			pg_size_pretty(pg_relation_size(i.oid)) as size,
			pg_get_indexdef(i.oid) as definition
		FROM pg_index ix
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = i.relam
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1
		  AND t.relname = $2
		GROUP BY i.oid, i.relname, ix.indisunique, ix.indisprimary, am.amname
		ORDER BY i.relname
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var indexes []map[string]interface{}
	for rows.Next() {
		var name, indexType, size, definition string
		var columns []string
		var isUnique, isPrimary bool
		if err := rows.Scan(&name, &columns, &isUnique, &isPrimary, &indexType, &size, &definition); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		indexes = append(indexes, map[string]interface{}{
			"name":       name,
			"columns":    columns,
			"is_unique":  isUnique,
			"is_primary": isPrimary,
			"type":       indexType,
			"size":       size,
			"definition": definition,
		})
	}

	result, _ := json.MarshalIndent(indexes, "", "  ")
	c.Data(http.StatusOK, "application/json", result)
}
