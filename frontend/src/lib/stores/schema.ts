import { writable, derived } from 'svelte/store';
import type { Schema, Table, View, Function, Sequence, CustomType, SchemaTreeNode } from '$lib/types';
import { schemaApi } from '$lib/api/client';
import { activeConnectionId } from './connections';

export const schemas = writable<Schema[]>([]);
export const tables = writable<Table[]>([]);
export const views = writable<View[]>([]);
export const functions = writable<Function[]>([]);
export const sequences = writable<Sequence[]>([]);
export const customTypes = writable<CustomType[]>([]);
export const isLoading = writable(false);
export const error = writable<string | null>(null);

export const expandedNodes = writable<Set<string>>(new Set());

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

		schemas.set(schemaList);
		tables.set(tableList);
		views.set(viewList);
		functions.set(functionList);
		sequences.set(sequenceList);
		customTypes.set(typeList);
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
	expandedNodes.set(new Set());
}

// Build a tree structure for the sidebar
export const schemaTree = derived(
	[schemas, tables, views, functions, sequences, customTypes],
	([$schemas, $tables, $views, $functions, $sequences, $customTypes]): SchemaTreeNode[] => {
		return $schemas.map((schema) => {
			const schemaTables = $tables.filter((t) => t.schema === schema.name);
			const schemaViews = $views.filter((v) => v.schema === schema.name);
			const schemaFunctions = $functions.filter((f) => f.schema === schema.name);
			const schemaSequences = $sequences.filter((s) => s.schema === schema.name);
			const schemaTypes = $customTypes.filter((t) => t.schema === schema.name);

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
