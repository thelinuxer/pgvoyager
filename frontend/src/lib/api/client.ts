import type {
	Connection,
	ConnectionRequest,
	Schema,
	Table,
	Column,
	Constraint,
	Index,
	ForeignKey,
	View,
	TableDataResponse,
	ForeignKeyPreview,
	QueryResult
} from '$lib/types';

const API_BASE = 'http://localhost:8081/api';

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE}${path}`, {
		...options,
		headers: {
			'Content-Type': 'application/json',
			...options?.headers
		}
	});

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

	listViews: (connId: string, schema?: string) => {
		const params = schema ? `?schema=${encodeURIComponent(schema)}` : '';
		return fetchAPI<View[]>(`/schema/${connId}/views${params}`);
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
		}
	) => {
		const params = new URLSearchParams();
		if (options?.page) params.set('page', String(options.page));
		if (options?.pageSize) params.set('pageSize', String(options.pageSize));
		if (options?.orderBy) params.set('orderBy', options.orderBy);
		if (options?.orderDir) params.set('orderDir', options.orderDir);

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
		)
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
