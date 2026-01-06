package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/thelinuxer/pgvoyager/internal/database"
	"github.com/thelinuxer/pgvoyager/internal/models"
	"github.com/gin-gonic/gin"
)

func getPool(c *gin.Context) (*database.ConnectionManager, string, bool) {
	connId := c.Param("connId")
	manager := database.GetManager()
	if !manager.IsConnected(connId) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not connected"})
		return nil, "", false
	}
	return manager, connId, true
}

func ListDatabases(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT
			d.datname as name,
			pg_catalog.pg_get_userbyid(d.datdba) as owner,
			pg_catalog.pg_encoding_to_char(d.encoding) as encoding,
			d.datcollate as collation,
			pg_catalog.pg_size_pretty(pg_catalog.pg_database_size(d.datname)) as size,
			(SELECT count(*) FROM pg_catalog.pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema')) as table_count
		FROM pg_catalog.pg_database d
		WHERE d.datistemplate = false
		ORDER BY d.datname
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var databases []models.Database
	for rows.Next() {
		var db models.Database
		if err := rows.Scan(&db.Name, &db.Owner, &db.Encoding, &db.Collation, &db.Size, &db.TableCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		databases = append(databases, db)
	}

	c.JSON(http.StatusOK, databases)
}

func ListSchemas(c *gin.Context) {
	manager, connId, ok := getPool(c)
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

	var schemas []models.Schema
	for rows.Next() {
		var s models.Schema
		if err := rows.Scan(&s.Name, &s.Owner, &s.TableCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		schemas = append(schemas, s)
	}

	c.JSON(http.StatusOK, schemas)
}

func ListTables(c *gin.Context) {
	manager, connId, ok := getPool(c)
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
			EXISTS(SELECT 1 FROM pg_constraint con WHERE con.conrelid = c.oid AND con.contype = 'p') as has_pk,
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

	var tables []models.Table
	for rows.Next() {
		var t models.Table
		if err := rows.Scan(&t.Schema, &t.Name, &t.Owner, &t.RowCount, &t.Size, &t.HasPK, &t.Comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tables = append(tables, t)
	}

	c.JSON(http.StatusOK, tables)
}

func GetTableInfo(c *gin.Context) {
	manager, connId, ok := getPool(c)
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
			EXISTS(SELECT 1 FROM pg_constraint con WHERE con.conrelid = c.oid AND con.contype = 'p') as has_pk,
			COALESCE(obj_description(c.oid), '') as comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r'
		  AND n.nspname = $1
		  AND c.relname = $2
	`

	var t models.Table
	err := pool.QueryRow(ctx, query, schema, table).Scan(
		&t.Schema, &t.Name, &t.Owner, &t.RowCount, &t.Size, &t.HasPK, &t.Comment,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	c.JSON(http.StatusOK, t)
}

func GetTableColumns(c *gin.Context) {
	manager, connId, ok := getPool(c)
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
			t.typname as udt_name,
			NOT a.attnotnull as is_nullable,
			pg_catalog.pg_get_expr(d.adbin, d.adrelid) as default_value,
			COALESCE(pk.is_pk, false) as is_primary_key,
			COALESCE(fk.is_fk, false) as is_foreign_key,
			fk.ref_schema,
			fk.ref_table,
			fk.ref_column,
			CASE WHEN a.atttypmod > 0 THEN a.atttypmod - 4 ELSE NULL END as max_length,
			COALESCE(col_description(c.oid, a.attnum), '') as comment
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_catalog.pg_type t ON t.oid = a.atttypid
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

	var columns []models.Column
	for rows.Next() {
		var col models.Column
		var refSchema, refTable, refColumn *string

		if err := rows.Scan(
			&col.Name, &col.Position, &col.DataType, &col.UDTName,
			&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsForeignKey,
			&refSchema, &refTable, &refColumn, &col.MaxLength, &col.Comment,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
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

	c.JSON(http.StatusOK, columns)
}

func GetTableConstraints(c *gin.Context) {
	manager, connId, ok := getPool(c)
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
			CASE con.contype
				WHEN 'p' THEN 'PRIMARY KEY'
				WHEN 'f' THEN 'FOREIGN KEY'
				WHEN 'u' THEN 'UNIQUE'
				WHEN 'c' THEN 'CHECK'
				WHEN 'x' THEN 'EXCLUSION'
			END as type,
			array_agg(a.attname ORDER BY array_position(con.conkey, a.attnum)) as columns,
			pg_get_constraintdef(con.oid) as definition,
			nf.nspname as ref_schema,
			cf.relname as ref_table,
			CASE WHEN con.contype = 'f' THEN
				array_agg(af.attname ORDER BY array_position(con.confkey, af.attnum))
			END as ref_columns
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(con.conkey)
		LEFT JOIN pg_class cf ON cf.oid = con.confrelid
		LEFT JOIN pg_namespace nf ON nf.oid = cf.relnamespace
		LEFT JOIN pg_attribute af ON af.attrelid = con.confrelid AND af.attnum = ANY(con.confkey)
		WHERE n.nspname = $1
		  AND c.relname = $2
		GROUP BY con.oid, con.conname, con.contype, nf.nspname, cf.relname
		ORDER BY con.contype, con.conname
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var constraints []models.Constraint
	for rows.Next() {
		var con models.Constraint
		var refSchema, refTable *string
		var refColumns []string

		if err := rows.Scan(
			&con.Name, &con.Type, &con.Columns, &con.Definition,
			&refSchema, &refTable, &refColumns,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if refSchema != nil {
			con.RefSchema = *refSchema
			con.RefTable = *refTable
			con.RefColumns = refColumns
		}

		constraints = append(constraints, con)
	}

	c.JSON(http.StatusOK, constraints)
}

func GetTableIndexes(c *gin.Context) {
	manager, connId, ok := getPool(c)
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

	var indexes []models.Index
	for rows.Next() {
		var idx models.Index
		if err := rows.Scan(
			&idx.Name, &idx.Columns, &idx.IsUnique, &idx.IsPrimary,
			&idx.Type, &idx.Size, &idx.Definition,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		indexes = append(indexes, idx)
	}

	c.JSON(http.StatusOK, indexes)
}

func GetForeignKeys(c *gin.Context) {
	manager, connId, ok := getPool(c)
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

	var fks []models.ForeignKey
	for rows.Next() {
		var fk models.ForeignKey
		if err := rows.Scan(
			&fk.Name, &fk.Columns, &fk.RefSchema, &fk.RefTable,
			&fk.RefColumns, &fk.OnUpdate, &fk.OnDelete,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fks = append(fks, fk)
	}

	c.JSON(http.StatusOK, fks)
}

// GetSchemaRelationships returns all foreign key relationships within a schema
// Used for ERD visualization
func GetSchemaRelationships(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := c.Param("schema")

	query := `
		SELECT
			n.nspname as source_schema,
			c.relname as source_table,
			array_agg(a.attname ORDER BY array_position(con.conkey, a.attnum)) as source_columns,
			nf.nspname as target_schema,
			cf.relname as target_table,
			array_agg(af.attname ORDER BY array_position(con.confkey, af.attnum)) as target_columns,
			con.conname as constraint_name,
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
		  AND (n.nspname = $1 OR nf.nspname = $1)
		GROUP BY con.oid, n.nspname, c.relname, nf.nspname, cf.relname, con.conname, con.confupdtype, con.confdeltype
		ORDER BY c.relname, con.conname
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var relationships []models.SchemaRelationship
	for rows.Next() {
		var rel models.SchemaRelationship
		if err := rows.Scan(
			&rel.SourceSchema, &rel.SourceTable, &rel.SourceColumns,
			&rel.TargetSchema, &rel.TargetTable, &rel.TargetColumns,
			&rel.ConstraintName, &rel.OnUpdate, &rel.OnDelete,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		relationships = append(relationships, rel)
	}

	c.JSON(http.StatusOK, relationships)
}

func ListViews(c *gin.Context) {
	manager, connId, ok := getPool(c)
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

	var views []models.View
	for rows.Next() {
		var v models.View
		if err := rows.Scan(&v.Schema, &v.Name, &v.Owner, &v.Definition, &v.Comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		views = append(views, v)
	}

	c.JSON(http.StatusOK, views)
}

func ListFunctions(c *gin.Context) {
	manager, connId, ok := getPool(c)
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
			pg_get_functiondef(p.oid) as definition,
			p.prokind = 'a' as is_aggregate,
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

	query += " ORDER BY n.nspname, p.proname"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var functions []models.Function
	for rows.Next() {
		var f models.Function
		if err := rows.Scan(
			&f.Schema, &f.Name, &f.Owner, &f.ReturnType, &f.Arguments,
			&f.Language, &f.Definition, &f.IsAggregate, &f.Comment,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		functions = append(functions, f)
	}

	c.JSON(http.StatusOK, functions)
}

func ListSequences(c *gin.Context) {
	manager, connId, ok := getPool(c)
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
			s.seqtypid::regtype::text as data_type,
			s.seqstart as start_value,
			s.seqmin as min_value,
			s.seqmax as max_value,
			s.seqincrement as increment,
			s.seqcache as cache_size,
			s.seqcycle as is_cycled
		FROM pg_catalog.pg_sequence s
		JOIN pg_catalog.pg_class c ON c.oid = s.seqrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname NOT LIKE 'pg_%'
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

	var sequences []models.Sequence
	for rows.Next() {
		var s models.Sequence
		if err := rows.Scan(
			&s.Schema, &s.Name, &s.Owner, &s.DataType, &s.StartValue,
			&s.MinValue, &s.MaxValue, &s.Increment, &s.CacheSize, &s.IsCycled,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sequences = append(sequences, s)
	}

	c.JSON(http.StatusOK, sequences)
}

func ListTypes(c *gin.Context) {
	manager, connId, ok := getPool(c)
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
			t.typname as name,
			pg_catalog.pg_get_userbyid(t.typowner) as owner,
			CASE t.typtype
				WHEN 'e' THEN 'enum'
				WHEN 'c' THEN 'composite'
				WHEN 'd' THEN 'domain'
				WHEN 'r' THEN 'range'
				ELSE 'other'
			END as type,
			CASE WHEN t.typtype = 'e' THEN
				array_agg(e.enumlabel ORDER BY e.enumsortorder)
			END as elements,
			COALESCE(obj_description(t.oid, 'pg_type'), '') as comment
		FROM pg_catalog.pg_type t
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		LEFT JOIN pg_catalog.pg_enum e ON e.enumtypid = t.oid
		WHERE t.typtype IN ('e', 'c', 'd', 'r')
		  AND n.nspname NOT LIKE 'pg_%'
		  AND n.nspname != 'information_schema'
	`

	args := []interface{}{}
	if schemaFilter != "" {
		query += " AND n.nspname = $1"
		args = append(args, schemaFilter)
	}

	query += " GROUP BY n.nspname, t.typname, t.typowner, t.typtype, t.oid ORDER BY n.nspname, t.typname"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var types []models.CustomType
	for rows.Next() {
		var t models.CustomType
		if err := rows.Scan(&t.Schema, &t.Name, &t.Owner, &t.Type, &t.Elements, &t.Comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		types = append(types, t)
	}

	c.JSON(http.StatusOK, types)
}

// TableColumns represents columns for a specific table
type TableColumns struct {
	Schema  string          `json:"schema"`
	Table   string          `json:"table"`
	Columns []models.Column `json:"columns"`
}

// GetAllColumns returns columns for all tables in a single request
// This is optimized for autocomplete to avoid N+1 queries
func GetAllColumns(c *gin.Context) {
	manager, connId, ok := getPool(c)
	if !ok {
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	query := `
		SELECT
			n.nspname as schema_name,
			c.relname as table_name,
			a.attname as name,
			a.attnum as position,
			pg_catalog.format_type(a.atttypid, a.atttypmod) as data_type,
			t.typname as udt_name,
			NOT a.attnotnull as is_nullable,
			pg_catalog.pg_get_expr(d.adbin, d.adrelid) as default_value,
			COALESCE(pk.is_pk, false) as is_primary_key,
			COALESCE(fk.is_fk, false) as is_foreign_key,
			fk.ref_schema,
			fk.ref_table,
			fk.ref_column,
			CASE WHEN a.atttypmod > 0 THEN a.atttypmod - 4 ELSE NULL END as max_length,
			COALESCE(col_description(c.oid, a.attnum), '') as comment
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_catalog.pg_type t ON t.oid = a.atttypid
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
		WHERE c.relkind IN ('r', 'p')
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY n.nspname, c.relname, a.attnum
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Group columns by schema.table
	tableColumnsMap := make(map[string]*TableColumns)

	for rows.Next() {
		var schemaName, tableName string
		var col models.Column
		var refSchema, refTable, refColumn *string

		if err := rows.Scan(
			&schemaName, &tableName,
			&col.Name, &col.Position, &col.DataType, &col.UDTName,
			&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsForeignKey,
			&refSchema, &refTable, &refColumn, &col.MaxLength, &col.Comment,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if col.IsForeignKey && refSchema != nil {
			col.FKReference = &models.FKRef{
				Schema: *refSchema,
				Table:  *refTable,
				Column: *refColumn,
			}
		}

		key := schemaName + "." + tableName
		if _, exists := tableColumnsMap[key]; !exists {
			tableColumnsMap[key] = &TableColumns{
				Schema:  schemaName,
				Table:   tableName,
				Columns: []models.Column{},
			}
		}
		tableColumnsMap[key].Columns = append(tableColumnsMap[key].Columns, col)
	}

	// Convert map to slice
	result := make([]TableColumns, 0, len(tableColumnsMap))
	for _, tc := range tableColumnsMap {
		result = append(result, *tc)
	}

	c.JSON(http.StatusOK, result)
}
