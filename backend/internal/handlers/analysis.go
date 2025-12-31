package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thelinuxer/pgvoyager/internal/database"
	"github.com/thelinuxer/pgvoyager/internal/models"
)

// RunAnalysis performs database health and optimization analysis
func RunAnalysis(c *gin.Context) {
	connId := c.Param("connId")
	manager := database.GetManager()
	if !manager.IsConnected(connId) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not connected"})
		return
	}

	pool, _ := manager.GetPool(connId)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := models.AnalysisResult{
		Categories: []models.AnalysisCategory{},
	}

	// Gather all issues
	indexIssues := analyzeIndexes(ctx, pool)
	tableIssues := analyzeTables(ctx, pool)
	constraintIssues := analyzeConstraints(ctx, pool)
	sequenceIssues := analyzeSequences(ctx, pool)
	performanceIssues := analyzePerformance(ctx, pool)

	// Build categories
	if len(indexIssues) > 0 {
		result.Categories = append(result.Categories, models.AnalysisCategory{
			Name:   "Index Health",
			Icon:   "zap",
			Issues: indexIssues,
		})
	}
	if len(tableIssues) > 0 {
		result.Categories = append(result.Categories, models.AnalysisCategory{
			Name:   "Table Health",
			Icon:   "table",
			Issues: tableIssues,
		})
	}
	if len(constraintIssues) > 0 {
		result.Categories = append(result.Categories, models.AnalysisCategory{
			Name:   "Constraints",
			Icon:   "link",
			Issues: constraintIssues,
		})
	}
	if len(sequenceIssues) > 0 {
		result.Categories = append(result.Categories, models.AnalysisCategory{
			Name:   "Sequences",
			Icon:   "hash",
			Issues: sequenceIssues,
		})
	}
	if len(performanceIssues) > 0 {
		result.Categories = append(result.Categories, models.AnalysisCategory{
			Name:   "Performance",
			Icon:   "activity",
			Issues: performanceIssues,
		})
	}

	// Calculate summary
	for _, cat := range result.Categories {
		for _, issue := range cat.Issues {
			switch issue.Severity {
			case "critical":
				result.Summary.Critical++
			case "warning":
				result.Summary.Warning++
			case "info":
				result.Summary.Info++
			}
		}
	}

	// Get database stats
	result.Stats = getDatabaseStats(ctx, pool)

	c.JSON(http.StatusOK, result)
}

func analyzeIndexes(ctx context.Context, pool *pgxpool.Pool) []models.AnalysisIssue {
	issues := []models.AnalysisIssue{}

	// Missing FK indexes
	query := `
		SELECT
			n.nspname || '.' || c.relname AS table_name,
			a.attname AS column_name,
			con.conname AS constraint_name,
			nf.nspname || '.' || cf.relname AS ref_table
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_class cf ON cf.oid = con.confrelid
		JOIN pg_namespace nf ON nf.oid = cf.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(con.conkey)
		WHERE con.contype = 'f'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		AND NOT EXISTS (
			SELECT 1 FROM pg_index i
			WHERE i.indrelid = c.oid
			AND a.attnum = ANY(i.indkey)
		)
		LIMIT 20
	`
	rows, err := pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName, columnName, constraintName, refTable string
			if err := rows.Scan(&tableName, &columnName, &constraintName, &refTable); err == nil {
				issues = append(issues, models.AnalysisIssue{
					Severity:    "warning",
					Title:       "Missing index on foreign key",
					Description: fmt.Sprintf("FK '%s' on column '%s' has no index", constraintName, columnName),
					Table:       tableName,
					Column:      columnName,
					Suggestion:  fmt.Sprintf("CREATE INDEX ON %s (%s);", tableName, columnName),
					Impact:      fmt.Sprintf("JOINs to %s require sequential scans", refTable),
				})
			}
		}
	}

	// Unused indexes (0 scans, not primary keys)
	query = `
		SELECT schemaname || '.' || relname AS table_name,
		       indexrelname AS index_name,
		       pg_size_pretty(pg_relation_size(indexrelid)) AS size
		FROM pg_stat_user_indexes
		WHERE idx_scan = 0
		AND indexrelname NOT LIKE '%_pkey'
		AND pg_relation_size(indexrelid) > 8192
		ORDER BY pg_relation_size(indexrelid) DESC
		LIMIT 20
	`
	rows, err = pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName, indexName, size string
			if err := rows.Scan(&tableName, &indexName, &size); err == nil {
				issues = append(issues, models.AnalysisIssue{
					Severity:    "info",
					Title:       "Unused index",
					Description: fmt.Sprintf("Index '%s' has never been used (size: %s)", indexName, size),
					Table:       tableName,
					Suggestion:  fmt.Sprintf("DROP INDEX %s;", indexName),
					Impact:      "Wastes disk space and slows down writes",
				})
			}
		}
	}

	// Duplicate indexes
	query = `
		SELECT
			n.nspname || '.' || ct.relname AS table_name,
			array_agg(ci.relname ORDER BY ci.relname) AS index_names,
			pg_get_indexdef(i.indexrelid) AS definition
		FROM pg_index i
		JOIN pg_class ct ON ct.oid = i.indrelid
		JOIN pg_class ci ON ci.oid = i.indexrelid
		JOIN pg_namespace n ON n.oid = ct.relnamespace
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		GROUP BY n.nspname, ct.relname, i.indkey, pg_get_indexdef(i.indexrelid)
		HAVING count(*) > 1
		LIMIT 10
	`
	rows, err = pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName string
			var indexNames []string
			var definition string
			if err := rows.Scan(&tableName, &indexNames, &definition); err == nil {
				issues = append(issues, models.AnalysisIssue{
					Severity:    "warning",
					Title:       "Duplicate indexes",
					Description: fmt.Sprintf("Indexes on same columns: %v", indexNames),
					Table:       tableName,
					Impact:      "Wastes space and slows writes with redundant indexes",
				})
			}
		}
	}

	return issues
}

func analyzeTables(ctx context.Context, pool *pgxpool.Pool) []models.AnalysisIssue {
	issues := []models.AnalysisIssue{}

	// Tables without primary key
	query := `
		SELECT n.nspname || '.' || c.relname AS table_name
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		AND NOT EXISTS (
			SELECT 1 FROM pg_constraint con
			WHERE con.conrelid = c.oid AND con.contype = 'p'
		)
		LIMIT 20
	`
	rows, err := pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err == nil {
				issues = append(issues, models.AnalysisIssue{
					Severity:    "warning",
					Title:       "Table without primary key",
					Description: "No primary key defined",
					Table:       tableName,
					Impact:      "May cause issues with replication and ORMs",
				})
			}
		}
	}

	// Table bloat (high dead tuples)
	query = `
		SELECT schemaname || '.' || relname AS table_name,
		       n_dead_tup,
		       n_live_tup,
		       ROUND(100.0 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 1) AS dead_pct
		FROM pg_stat_user_tables
		WHERE n_dead_tup > 10000
		AND 100.0 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0) > 10
		ORDER BY n_dead_tup DESC
		LIMIT 10
	`
	rows, err = pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName string
			var deadTup, liveTup int64
			var deadPct float64
			if err := rows.Scan(&tableName, &deadTup, &liveTup, &deadPct); err == nil {
				issues = append(issues, models.AnalysisIssue{
					Severity:    "warning",
					Title:       "Table bloat",
					Description: fmt.Sprintf("%.1f%% dead tuples (%d dead rows)", deadPct, deadTup),
					Table:       tableName,
					Suggestion:  fmt.Sprintf("VACUUM ANALYZE %s;", tableName),
					Impact:      "Wasted disk space and slower queries",
				})
			}
		}
	}

	// Stale statistics (never analyzed or very old)
	query = `
		SELECT schemaname || '.' || relname AS table_name,
		       last_analyze,
		       last_autoanalyze
		FROM pg_stat_user_tables
		WHERE n_live_tup > 1000
		AND (last_analyze IS NULL AND last_autoanalyze IS NULL)
		   OR (COALESCE(last_analyze, last_autoanalyze) < NOW() - INTERVAL '7 days')
		LIMIT 10
	`
	rows, err = pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName string
			var lastAnalyze, lastAutoanalyze *time.Time
			if err := rows.Scan(&tableName, &lastAnalyze, &lastAutoanalyze); err == nil {
				desc := "Never analyzed"
				if lastAnalyze != nil || lastAutoanalyze != nil {
					desc = "Statistics are stale (>7 days old)"
				}
				issues = append(issues, models.AnalysisIssue{
					Severity:    "info",
					Title:       "Stale table statistics",
					Description: desc,
					Table:       tableName,
					Suggestion:  fmt.Sprintf("ANALYZE %s;", tableName),
					Impact:      "Query planner may choose suboptimal plans",
				})
			}
		}
	}

	return issues
}

func analyzeConstraints(ctx context.Context, pool *pgxpool.Pool) []models.AnalysisIssue {
	issues := []models.AnalysisIssue{}
	// Constraints analysis is typically covered by FK index check
	// Could add check for invalid constraints if needed
	return issues
}

func analyzeSequences(ctx context.Context, pool *pgxpool.Pool) []models.AnalysisIssue {
	issues := []models.AnalysisIssue{}

	// Sequences approaching exhaustion
	query := `
		SELECT schemaname || '.' || sequencename AS seq_name,
		       last_value,
		       max_value,
		       ROUND(100.0 * last_value / max_value, 2) AS pct_used
		FROM pg_sequences
		WHERE last_value IS NOT NULL
		AND max_value > 0
		AND 100.0 * last_value / max_value > 50
		ORDER BY pct_used DESC
		LIMIT 10
	`
	rows, err := pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var seqName string
			var lastValue, maxValue int64
			var pctUsed float64
			if err := rows.Scan(&seqName, &lastValue, &maxValue, &pctUsed); err == nil {
				severity := "info"
				if pctUsed > 90 {
					severity = "critical"
				} else if pctUsed > 75 {
					severity = "warning"
				}
				issues = append(issues, models.AnalysisIssue{
					Severity:    severity,
					Title:       "Sequence approaching limit",
					Description: fmt.Sprintf("%.1f%% used (%d of %d)", pctUsed, lastValue, maxValue),
					Table:       seqName,
					Impact:      "Will cause errors when exhausted",
				})
			}
		}
	}

	return issues
}

func analyzePerformance(ctx context.Context, pool *pgxpool.Pool) []models.AnalysisIssue {
	issues := []models.AnalysisIssue{}

	// Low cache hit ratio
	query := `
		SELECT ROUND(100.0 * sum(blks_hit) / NULLIF(sum(blks_hit) + sum(blks_read), 0), 2) AS ratio
		FROM pg_stat_database
		WHERE datname = current_database()
	`
	var cacheRatio *float64
	err := pool.QueryRow(ctx, query).Scan(&cacheRatio)
	if err == nil && cacheRatio != nil && *cacheRatio < 90 {
		issues = append(issues, models.AnalysisIssue{
			Severity:    "warning",
			Title:       "Low cache hit ratio",
			Description: fmt.Sprintf("Buffer cache hit ratio is %.1f%%", *cacheRatio),
			Impact:      "Queries are reading from disk frequently",
			Suggestion:  "Consider increasing shared_buffers",
		})
	}

	// Long running queries (> 5 minutes)
	query = `
		SELECT pid,
		       usename,
		       EXTRACT(EPOCH FROM (now() - query_start))::int AS duration_secs,
		       LEFT(query, 100) AS query_preview
		FROM pg_stat_activity
		WHERE state = 'active'
		AND query NOT LIKE '%pg_stat_activity%'
		AND now() - query_start > INTERVAL '5 minutes'
		LIMIT 5
	`
	rows, err := pool.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pid int
			var usename string
			var durationSecs int
			var queryPreview string
			if err := rows.Scan(&pid, &usename, &durationSecs, &queryPreview); err == nil {
				issues = append(issues, models.AnalysisIssue{
					Severity:    "warning",
					Title:       "Long-running query",
					Description: fmt.Sprintf("PID %d running for %d seconds: %s...", pid, durationSecs, queryPreview),
					Impact:      "May be holding locks or consuming resources",
					Suggestion:  fmt.Sprintf("SELECT pg_cancel_backend(%d);", pid),
				})
			}
		}
	}

	return issues
}

func getDatabaseStats(ctx context.Context, pool *pgxpool.Pool) models.DatabaseStats {
	stats := models.DatabaseStats{}

	// Database size
	query := `SELECT pg_size_pretty(pg_database_size(current_database()))`
	pool.QueryRow(ctx, query).Scan(&stats.DatabaseSize)

	// Table count
	query = `SELECT count(*) FROM pg_stat_user_tables`
	pool.QueryRow(ctx, query).Scan(&stats.TableCount)

	// Index count
	query = `SELECT count(*) FROM pg_stat_user_indexes`
	pool.QueryRow(ctx, query).Scan(&stats.IndexCount)

	// Cache hit ratio
	query = `
		SELECT COALESCE(ROUND(100.0 * sum(blks_hit) / NULLIF(sum(blks_hit) + sum(blks_read), 0), 2), 0)
		FROM pg_stat_database
		WHERE datname = current_database()
	`
	pool.QueryRow(ctx, query).Scan(&stats.CacheHitRatio)

	// Active connections
	query = `SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()`
	pool.QueryRow(ctx, query).Scan(&stats.ActiveConnections)

	return stats
}
