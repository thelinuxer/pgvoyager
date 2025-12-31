export interface Connection {
	id: string;
	name: string;
	host: string;
	port: number;
	database: string;
	username: string;
	password?: string;
	sslMode: string;
	isConnected: boolean;
	createdAt: string;
	updatedAt: string;
}

export interface ConnectionRequest {
	name: string;
	host: string;
	port: number;
	database: string;
	username: string;
	password: string;
	sslMode: string;
}

export interface Schema {
	name: string;
	owner: string;
	tableCount: number;
}

export interface Table {
	schema: string;
	name: string;
	owner: string;
	rowCount: number;
	size: string;
	hasPk: boolean;
	comment?: string;
}

export interface Column {
	name: string;
	position: number;
	dataType: string;
	udtName: string;
	isNullable: boolean;
	defaultValue?: string;
	isPrimaryKey: boolean;
	isForeignKey: boolean;
	fkReference?: FKRef;
	maxLength?: number;
	comment?: string;
}

export interface FKRef {
	schema: string;
	table: string;
	column: string;
}

export interface ColumnInfo {
	name: string;
	dataType: string;
	isPrimaryKey: boolean;
	isForeignKey: boolean;
	fkReference?: FKRef;
}

export interface Constraint {
	name: string;
	type: string;
	columns: string[];
	definition: string;
	refSchema?: string;
	refTable?: string;
	refColumns?: string[];
}

export interface Index {
	name: string;
	columns: string[];
	isUnique: boolean;
	isPrimary: boolean;
	type: string;
	size: string;
	definition: string;
}

export interface ForeignKey {
	name: string;
	columns: string[];
	refSchema: string;
	refTable: string;
	refColumns: string[];
	onUpdate: string;
	onDelete: string;
}

// SchemaRelationship represents a foreign key relationship for ERD visualization
export interface SchemaRelationship {
	sourceSchema: string;
	sourceTable: string;
	sourceColumns: string[];
	targetSchema: string;
	targetTable: string;
	targetColumns: string[];
	constraintName: string;
	onUpdate: string;
	onDelete: string;
}

export interface View {
	schema: string;
	name: string;
	owner: string;
	definition: string;
	comment?: string;
}

export interface Function {
	schema: string;
	name: string;
	owner: string;
	returnType: string;
	arguments: string;
	language: string;
	definition: string;
	isAggregate: boolean;
	comment?: string;
}

export interface Sequence {
	schema: string;
	name: string;
	owner: string;
	dataType: string;
	startValue: number;
	minValue: number;
	maxValue: number;
	increment: number;
	cacheSize: number;
	isCycled: boolean;
	lastValue?: number;
}

export interface CustomType {
	schema: string;
	name: string;
	owner: string;
	type: string; // enum, composite, domain, range
	elements?: string[];
	comment?: string;
}

export interface TableDataResponse {
	columns: ColumnInfo[];
	rows: Record<string, unknown>[];
	totalRows: number;
	page: number;
	pageSize: number;
	totalPages: number;
}

export interface ForeignKeyPreview {
	schema: string;
	table: string;
	columns: ColumnInfo[];
	row: Record<string, unknown>;
}

export interface QueryResult {
	columns: ColumnInfo[];
	rows: Record<string, unknown>[];
	rowCount: number;
	duration: number;
	error?: string;
	errorPosition?: number; // 1-based character position in SQL
	errorHint?: string;
	errorDetail?: string;
}

export interface SavedQuery {
	id: string;
	name: string;
	sql: string;
	connectionId?: string;
	description?: string;
	createdAt: string;
	updatedAt: string;
}

export interface SavedQueryRequest {
	name: string;
	sql: string;
	connectionId?: string;
	description?: string;
}

// CRUD operations
export interface InsertRowRequest {
	data: Record<string, unknown>;
}

export interface UpdateRowRequest {
	primaryKey: Record<string, unknown>;
	data: Record<string, unknown>;
}

export interface DeleteRowRequest {
	primaryKey: Record<string, unknown>;
}

export interface CrudResponse {
	success: boolean;
	rowsAffected: number;
	message?: string;
	insertedRow?: Record<string, unknown>;
}

export type TabType = 'table' | 'query' | 'view' | 'function' | 'sequence' | 'type' | 'erd';

// ERD navigation location
export interface ERDLocation {
	schema: string;
	centeredTable?: string; // If undefined, show full schema view
}

export interface TableLocation {
	schema: string;
	table: string;
	filter?: {
		column: string;
		value: string;
	};
	sort?: {
		column: string;
		direction: 'ASC' | 'DESC';
	};
	limit?: number;
}

export interface Tab {
	id: string;
	type: TabType;
	title: string;
	schema?: string;
	table?: string;
	view?: string;
	functionName?: string;
	sequenceName?: string;
	typeName?: string;
	isPinned: boolean;
	data?: TableDataResponse | QueryResult;
	// Navigation stack for table tabs
	navigationStack?: TableLocation[];
	navigationIndex?: number;
	// For query tabs
	initialSql?: string;
	// For ERD tabs
	erdNavigationStack?: ERDLocation[];
	erdNavigationIndex?: number;
}

export interface SchemaTreeNode {
	type: 'schema' | 'folder' | 'table' | 'view' | 'function' | 'sequence' | 'type';
	name: string;
	schema?: string;
	children?: SchemaTreeNode[];
	isExpanded?: boolean;
	data?: Table | View | Function | Sequence | CustomType;
}
