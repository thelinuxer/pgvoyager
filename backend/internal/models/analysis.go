package models

// AnalysisResult is the complete response from database analysis
type AnalysisResult struct {
	Summary    AnalysisSummary    `json:"summary"`
	Categories []AnalysisCategory `json:"categories"`
	Stats      DatabaseStats      `json:"stats"`
}

// AnalysisSummary counts issues by severity
type AnalysisSummary struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
	Ok       int `json:"ok"`
}

// AnalysisCategory groups related issues
type AnalysisCategory struct {
	Name   string          `json:"name"`
	Icon   string          `json:"icon"`
	Issues []AnalysisIssue `json:"issues"`
}

// AnalysisIssue represents a single finding
type AnalysisIssue struct {
	Severity    string `json:"severity"` // "critical", "warning", "info"
	Title       string `json:"title"`
	Description string `json:"description"`
	Table       string `json:"table,omitempty"`
	Column      string `json:"column,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
	Impact      string `json:"impact,omitempty"`
}

// DatabaseStats contains overall database metrics
type DatabaseStats struct {
	DatabaseSize      string  `json:"databaseSize"`
	TableCount        int     `json:"tableCount"`
	IndexCount        int     `json:"indexCount"`
	CacheHitRatio     float64 `json:"cacheHitRatio"`
	ActiveConnections int     `json:"activeConnections"`
}
