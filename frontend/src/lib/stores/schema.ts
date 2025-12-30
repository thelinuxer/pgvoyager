import { writable, derived, get } from 'svelte/store';
import type { Schema, Table, View, Function, Sequence, CustomType, SchemaTreeNode, Column } from '$lib/types';
import { schemaApi } from '$lib/api/client';
import { activeConnectionId } from './connections';
import type { ColumnInfo } from '$lib/utils/sqlAutocomplete';

export const schemas = writable<Schema[]>([]);
export const tables = writable<Table[]>([]);
export const views = writable<View[]>([]);
export const functions = writable<Function[]>([]);
export const sequences = writable<Sequence[]>([]);
export const customTypes = writable<CustomType[]>([]);
export const isLoading = writable(false);
export const error = writable<string | null>(null);

export const expandedNodes = writable<Set<string>>(new Set());

// Store for caching table columns (key: "schema.table")
export const tableColumns = writable<Map<string, Column[]>>(new Map());

// Derived store that flattens all columns for autocomplete
export const allColumns = derived(tableColumns, ($tableColumns): ColumnInfo[] => {
	const result: ColumnInfo[] = [];
	for (const [key, columns] of $tableColumns.entries()) {
		const [schema, tableName] = key.split('.');
		for (const col of columns) {
			result.push({
				name: col.name,
				tableName,
				schema,
				dataType: col.dataType
			});
		}
	}
	return result;
});

// Load columns for a specific table and cache them
export async function loadTableColumns(connId: string, schema: string, table: string): Promise<Column[]> {
	const key = `${schema}.${table}`;
	const cached = get(tableColumns).get(key);
	if (cached) {
		return cached;
	}

	try {
		const columns = await schemaApi.getTableColumns(connId, schema, table);
		tableColumns.update((map) => {
			const newMap = new Map(map);
			newMap.set(key, columns);
			return newMap;
		});
		return columns;
	} catch (e) {
		console.error(`Failed to load columns for ${key}:`, e);
		return [];
	}
}

// Preload columns for all tables (for autocomplete)
export async function preloadAllColumns(connId: string): Promise<void> {
	const tableList = get(tables);
	const promises = tableList.map((t) => loadTableColumns(connId, t.schema, t.name));
	await Promise.all(promises);
}

export async function loadSchema(connId: string) {
	isLoading.set(true);
	error.set(null);

	try {
		const [schemaList, tableList, viewList, functionList, sequenceList, typeList] = await Promise.all([
			schemaApi.listSchemas(connId),
			schemaApi.listTables(connId),
			schemaApi.listViews(connId),
			schemaApi.listFunctions(connId),
			schemaApi.listSequences(connId),
			schemaApi.listTypes(connId)
		]);

		// Handle null responses from API by defaulting to empty arrays
		schemas.set(schemaList || []);
		tables.set(tableList || []);
		views.set(viewList || []);
		functions.set(functionList || []);
		sequences.set(sequenceList || []);
		customTypes.set(typeList || []);

		// Preload columns for autocomplete (in background, don't block)
		preloadAllColumns(connId).catch((e) => {
			console.error('Failed to preload columns for autocomplete:', e);
		});
	} catch (e) {
		error.set(e instanceof Error ? e.message : 'Failed to load schema');
	} finally {
		isLoading.set(false);
	}
}

export function clearSchema() {
	schemas.set([]);
	tables.set([]);
	views.set([]);
	functions.set([]);
	sequences.set([]);
	customTypes.set([]);
	tableColumns.set(new Map());
	expandedNodes.set(new Set());
}

// Build a tree structure for the sidebar
export const schemaTree = derived(
	[schemas, tables, views, functions, sequences, customTypes],
	([$schemas, $tables, $views, $functions, $sequences, $customTypes]): SchemaTreeNode[] => {
		// Safety: ensure arrays are not null
		const safeSchemas = $schemas || [];
		const safeTables = $tables || [];
		const safeViews = $views || [];
		const safeFunctions = $functions || [];
		const safeSequences = $sequences || [];
		const safeTypes = $customTypes || [];

		return safeSchemas.map((schema) => {
			const schemaTables = safeTables.filter((t) => t.schema === schema.name);
			const schemaViews = safeViews.filter((v) => v.schema === schema.name);
			const schemaFunctions = safeFunctions.filter((f) => f.schema === schema.name);
			const schemaSequences = safeSequences.filter((s) => s.schema === schema.name);
			const schemaTypes = safeTypes.filter((t) => t.schema === schema.name);

			const children: SchemaTreeNode[] = [];

			if (schemaTables.length > 0) {
				children.push({
					type: 'folder',
					name: 'Tables',
					schema: schema.name,
					children: schemaTables.map((t) => ({
						type: 'table',
						name: t.name,
						schema: t.schema,
						data: t
					}))
				});
			}

			if (schemaViews.length > 0) {
				children.push({
					type: 'folder',
					name: 'Views',
					schema: schema.name,
					children: schemaViews.map((v) => ({
						type: 'view',
						name: v.name,
						schema: v.schema,
						data: v
					}))
				});
			}

			if (schemaFunctions.length > 0) {
				children.push({
					type: 'folder',
					name: 'Functions',
					schema: schema.name,
					children: schemaFunctions.map((f) => ({
						type: 'function',
						name: f.name,
						schema: f.schema,
						data: f
					}))
				});
			}

			if (schemaSequences.length > 0) {
				children.push({
					type: 'folder',
					name: 'Sequences',
					schema: schema.name,
					children: schemaSequences.map((s) => ({
						type: 'sequence',
						name: s.name,
						schema: s.schema,
						data: s
					}))
				});
			}

			if (schemaTypes.length > 0) {
				children.push({
					type: 'folder',
					name: 'Types',
					schema: schema.name,
					children: schemaTypes.map((t) => ({
						type: 'type',
						name: t.name,
						schema: t.schema,
						data: t
					}))
				});
			}

			return {
				type: 'schema',
				name: schema.name,
				children
			} as SchemaTreeNode;
		});
	}
);

export function toggleNode(nodeKey: string) {
	expandedNodes.update((set) => {
		const newSet = new Set(set);
		if (newSet.has(nodeKey)) {
			newSet.delete(nodeKey);
		} else {
			newSet.add(nodeKey);
		}
		return newSet;
	});
}

// Auto-load schema when connection changes
let currentConnId: string | null = null;
activeConnectionId.subscribe((connId) => {
	if (connId && connId !== currentConnId) {
		currentConnId = connId;
		loadSchema(connId);
	} else if (!connId) {
		currentConnId = null;
		clearSchema();
	}
});
