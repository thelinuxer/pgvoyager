package models

type Database struct {
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	Encoding   string `json:"encoding"`
	Collation  string `json:"collation"`
	Size       string `json:"size"`
	TableCount int    `json:"tableCount"`
}

type Schema struct {
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	TableCount int    `json:"tableCount"`
}

type Table struct {
	Schema       string `json:"schema"`
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	RowCount     int64  `json:"rowCount"`
	Size         string `json:"size"`
	HasPK        bool   `json:"hasPk"`
	Comment      string `json:"comment,omitempty"`
}

type Column struct {
	Name         string  `json:"name"`
	Position     int     `json:"position"`
	DataType     string  `json:"dataType"`
	UDTName      string  `json:"udtName"`
	IsNullable   bool    `json:"isNullable"`
	DefaultValue *string `json:"defaultValue,omitempty"`
	IsPrimaryKey bool    `json:"isPrimaryKey"`
	IsForeignKey bool    `json:"isForeignKey"`
	FKReference  *FKRef  `json:"fkReference,omitempty"`
	MaxLength    *int    `json:"maxLength,omitempty"`
	Comment      string  `json:"comment,omitempty"`
}

type FKRef struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

type Constraint struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Columns    []string `json:"columns"`
	Definition string   `json:"definition"`
	RefSchema  string   `json:"refSchema,omitempty"`
	RefTable   string   `json:"refTable,omitempty"`
	RefColumns []string `json:"refColumns,omitempty"`
}

type Index struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	IsUnique   bool     `json:"isUnique"`
	IsPrimary  bool     `json:"isPrimary"`
	Type       string   `json:"type"`
	Size       string   `json:"size"`
	Definition string   `json:"definition"`
}

type ForeignKey struct {
	Name          string   `json:"name"`
	Columns       []string `json:"columns"`
	RefSchema     string   `json:"refSchema"`
	RefTable      string   `json:"refTable"`
	RefColumns    []string `json:"refColumns"`
	OnUpdate      string   `json:"onUpdate"`
	OnDelete      string   `json:"onDelete"`
}

type View struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	Definition string `json:"definition"`
	Comment    string `json:"comment,omitempty"`
}

type Function struct {
	Schema       string   `json:"schema"`
	Name         string   `json:"name"`
	Owner        string   `json:"owner"`
	ReturnType   string   `json:"returnType"`
	Arguments    string   `json:"arguments"`
	Language     string   `json:"language"`
	Definition   string   `json:"definition"`
	IsAggregate  bool     `json:"isAggregate"`
	Comment      string   `json:"comment,omitempty"`
}

type Sequence struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	DataType   string `json:"dataType"`
	StartValue int64  `json:"startValue"`
	MinValue   int64  `json:"minValue"`
	MaxValue   int64  `json:"maxValue"`
	Increment  int64  `json:"increment"`
	CacheSize  int64  `json:"cacheSize"`
	IsCycled   bool   `json:"isCycled"`
	LastValue  *int64 `json:"lastValue,omitempty"`
}

type CustomType struct {
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Type     string `json:"type"` // enum, composite, domain, range
	Elements []string `json:"elements,omitempty"` // for enums
	Comment  string `json:"comment,omitempty"`
}
