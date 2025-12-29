import { writable, derived } from 'svelte/store';
import type { Schema, Table, View, SchemaTreeNode } from '$lib/types';
import { schemaApi } from '$lib/api/client';
import { activeConnectionId } from './connections';

export const schemas = writable<Schema[]>([]);
export const tables = writable<Table[]>([]);
export const views = writable<View[]>([]);
export const isLoading = writable(false);
export const error = writable<string | null>(null);

export const expandedNodes = writable<Set<string>>(new Set());

export async function loadSchema(connId: string) {
	isLoading.set(true);
	error.set(null);

	try {
		const [schemaList, tableList, viewList] = await Promise.all([
			schemaApi.listSchemas(connId),
			schemaApi.listTables(connId),
			schemaApi.listViews(connId)
		]);

		schemas.set(schemaList);
		tables.set(tableList);
		views.set(viewList);
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
	expandedNodes.set(new Set());
}

// Build a tree structure for the sidebar
export const schemaTree = derived(
	[schemas, tables, views],
	([$schemas, $tables, $views]): SchemaTreeNode[] => {
		return $schemas.map((schema) => {
			const schemaTables = $tables.filter((t) => t.schema === schema.name);
			const schemaViews = $views.filter((v) => v.schema === schema.name);

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
