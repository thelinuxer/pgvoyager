import type {
	Connection,
	ConnectionRequest,
	Schema,
	Table,
	Column,
	Constraint,
	Index,
	ForeignKey,
	SchemaRelationship,
	View,
	Function,
	Sequence,
	CustomType,
	TableDataResponse,
	ForeignKeyPreview,
	QueryResult,
	SavedQuery,
	SavedQueryRequest,
	InsertRowRequest,
	UpdateRowRequest,
	DeleteRowRequest,
	CrudResponse
} from '$lib/types';

// In production, the frontend is served from the same origin as the API
// In development, we need to specify the backend URL
function getApiBase(): string {
	if (typeof window === 'undefined') return '/api';

	// Development mode (Vite dev server on port 5173)
	if (window.location.port === '5173') {
		return 'http://localhost:5137/api';
	}

	// Production mode - use same origin (relative URL)
	return '/api';
}

const API_BASE = getApiBase();

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
	let response: Response;

	try {
		response = await fetch(`${API_BASE}${path}`, {
			...options,
			headers: {
				'Content-Type': 'application/json',
				...options?.headers
			}
		});
	} catch (e) {
		// Network error (server unreachable, CORS, etc.)
		if (e instanceof TypeError && e.message.includes('fetch')) {
			throw new Error('Unable to connect to server. Please check if the backend is running.');
		}
		throw new Error(e instanceof Error ? e.message : 'Network error');
	}

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: 'Unknown error' }));
		throw new Error(error.error || `HTTP ${response.status}`);
	}

	return response.json();
}

// Connection API
export const connectionApi = {
	list: () => fetchAPI<Connection[]>('/connections'),

	get: (id: string) => fetchAPI<Connection>(`/connections/${id}`),

	create: (data: ConnectionRequest) =>
		fetchAPI<Connection>('/connections', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	update: (id: string, data: ConnectionRequest) =>
		fetchAPI<Connection>(`/connections/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}),

	delete: (id: string) =>
		fetchAPI<void>(`/connections/${id}`, {
			method: 'DELETE'
		}),

	test: (data: Omit<ConnectionRequest, 'name'>) =>
		fetchAPI<{ success: boolean; message: string }>('/connections/test', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	connect: (id: string) =>
		fetchAPI<{ message: string }>(`/connections/${id}/connect`, {
			method: 'POST'
		}),

	disconnect: (id: string) =>
		fetchAPI<{ message: string }>(`/connections/${id}/disconnect`, {
			method: 'POST'
		})
};

// Schema API
export const schemaApi = {
	listSchemas: (connId: string) => fetchAPI<Schema[]>(`/schema/${connId}/schemas`),

	listTables: (connId: string, schema?: string) => {
		const params = schema ? `?schema=${encodeURIComponent(schema)}` : '';
		return fetchAPI<Table[]>(`/schema/${connId}/tables${params}`);
	},

	getTableInfo: (connId: string, schema: string, table: string) =>
		fetchAPI<Table>(`/schema/${connId}/tables/${schema}/${table}`),

	getTableColumns: (connId: string, schema: string, table: string) =>
		fetchAPI<Column[]>(`/schema/${connId}/tables/${schema}/${table}/columns`),

	getTableConstraints: (connId: string, schema: string, table: string) =>
		fetchAPI<Constraint[]>(`/schema/${connId}/tables/${schema}/${table}/constraints`),

	getTableIndexes: (connId: string, schema: string, table: string) =>
		fetchAPI<Index[]>(`/schema/${connId}/tables/${schema}/${table}/indexes`),

	getForeignKeys: (connId: string, schema: string, table: string) =>
		fetchAPI<ForeignKey[]>(`/schema/${connId}/tables/${schema}/${table}/foreign-keys`),

	getSchemaRelationships: (connId: string, schema: string) =>
		fetchAPI<SchemaRelationship[]>(`/schema/${connId}/schemas/${schema}/relationships`),

	listViews: (connId: string, schema?: string) => {
		const params = schema ? `?schema=${encodeURIComponent(schema)}` : '';
		return fetchAPI<View[]>(`/schema/${connId}/views${params}`);
	},

	listFunctions: (connId: string, schema?: string) => {
		const params = schema ? `?schema=${encodeURIComponent(schema)}` : '';
		return fetchAPI<Function[]>(`/schema/${connId}/functions${params}`);
	},

	listSequences: (connId: string, schema?: string) => {
		const params = schema ? `?schema=${encodeURIComponent(schema)}` : '';
		return fetchAPI<Sequence[]>(`/schema/${connId}/sequences${params}`);
	},

	listTypes: (connId: string, schema?: string) => {
		const params = schema ? `?schema=${encodeURIComponent(schema)}` : '';
		return fetchAPI<CustomType[]>(`/schema/${connId}/types${params}`);
	}
};

// Data API
export const dataApi = {
	getTableData: (
		connId: string,
		schema: string,
		table: string,
		options?: {
			page?: number;
			pageSize?: number;
			orderBy?: string;
			orderDir?: 'ASC' | 'DESC';
			filterColumn?: string;
			filterValue?: string;
		}
	) => {
		const params = new URLSearchParams();
		if (options?.page) params.set('page', String(options.page));
		if (options?.pageSize) params.set('pageSize', String(options.pageSize));
		if (options?.orderBy) params.set('orderBy', options.orderBy);
		if (options?.orderDir) params.set('orderDir', options.orderDir);
		if (options?.filterColumn) params.set('filterColumn', options.filterColumn);
		if (options?.filterValue) params.set('filterValue', options.filterValue);

		const queryString = params.toString();
		return fetchAPI<TableDataResponse>(
			`/data/${connId}/tables/${schema}/${table}${queryString ? '?' + queryString : ''}`
		);
	},

	getRowCount: (connId: string, schema: string, table: string) =>
		fetchAPI<{ count: number }>(`/data/${connId}/tables/${schema}/${table}/count`),

	getForeignKeyPreview: (connId: string, schema: string, table: string, column: string, value: string) =>
		fetchAPI<ForeignKeyPreview>(
			`/data/${connId}/fk-preview/${schema}/${table}/${column}/${encodeURIComponent(value)}`
		),

	// CRUD operations
	insertRow: (connId: string, schema: string, table: string, data: InsertRowRequest) =>
		fetchAPI<CrudResponse>(`/data/${connId}/tables/${schema}/${table}/rows`, {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	updateRow: (connId: string, schema: string, table: string, data: UpdateRowRequest) =>
		fetchAPI<CrudResponse>(`/data/${connId}/tables/${schema}/${table}/rows`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}),

	deleteRow: (connId: string, schema: string, table: string, data: DeleteRowRequest) =>
		fetchAPI<CrudResponse>(`/data/${connId}/tables/${schema}/${table}/rows`, {
			method: 'DELETE',
			body: JSON.stringify(data)
		}),

	dropTable: (connId: string, schema: string, table: string, cascade?: boolean) =>
		fetchAPI<{ success: boolean; message: string }>(`/data/${connId}/tables/${schema}/${table}`, {
			method: 'DELETE',
			body: JSON.stringify({ cascade: cascade ?? false })
		})
};

// Query API
export const queryApi = {
	execute: (connId: string, sql: string, params?: unknown[]) =>
		fetchAPI<QueryResult>(`/query/${connId}/execute`, {
			method: 'POST',
			body: JSON.stringify({ sql, params })
		}),

	explain: (connId: string, sql: string, params?: unknown[]) =>
		fetchAPI<{ plan: string; duration: number }>(`/query/${connId}/explain`, {
			method: 'POST',
			body: JSON.stringify({ sql, params })
		})
};

// Saved Queries API
export const savedQueryApi = {
	list: () => fetchAPI<SavedQuery[]>('/queries'),

	get: (id: string) => fetchAPI<SavedQuery>(`/queries/${id}`),

	create: (data: SavedQueryRequest) =>
		fetchAPI<SavedQuery>('/queries', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	update: (id: string, data: SavedQueryRequest) =>
		fetchAPI<SavedQuery>(`/queries/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}),

	delete: (id: string) =>
		fetchAPI<void>(`/queries/${id}`, {
			method: 'DELETE'
		})
};

// Update API
export interface VersionResponse {
	version: string;
}

export interface UpdateCheckResponse {
	currentVersion: string;
	latestVersion: string;
	hasUpdate: boolean;
	releaseUrl: string;
}

export const updateApi = {
	getVersion: () => fetchAPI<VersionResponse>('/version'),

	checkUpdate: () => fetchAPI<UpdateCheckResponse>('/update/check')
};
