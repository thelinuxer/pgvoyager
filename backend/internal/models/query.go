package models

type QueryRequest struct {
	SQL    string        `json:"sql" binding:"required"`
	Params []interface{} `json:"params,omitempty"`
}

type QueryResult struct {
	Columns  []ColumnInfo      `json:"columns"`
	Rows     []map[string]any  `json:"rows"`
	RowCount int               `json:"rowCount"`
	Duration float64           `json:"duration"` // milliseconds
	Error    string            `json:"error,omitempty"`
}

type ColumnInfo struct {
	Name         string  `json:"name"`
	DataType     string  `json:"dataType"`
	IsPrimaryKey bool    `json:"isPrimaryKey"`
	IsForeignKey bool    `json:"isForeignKey"`
	FKReference  *FKRef  `json:"fkReference,omitempty"`
}

type TableDataRequest struct {
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"pageSize" binding:"min=1,max=1000"`
	OrderBy   string `form:"orderBy"`
	OrderDir  string `form:"orderDir"`
	Filter    string `form:"filter"`
}

type TableDataResponse struct {
	Columns    []ColumnInfo     `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	TotalRows  int64            `json:"totalRows"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int              `json:"totalPages"`
}

type ForeignKeyPreview struct {
	Schema     string           `json:"schema"`
	Table      string           `json:"table"`
	Columns    []ColumnInfo     `json:"columns"`
	Row        map[string]any   `json:"row"`
}

type ExplainResult struct {
	Plan      string  `json:"plan"`
	Duration  float64 `json:"duration"`
}
