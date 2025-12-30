import { writable } from 'svelte/store';
import type { SavedQuery, SavedQueryRequest } from '$lib/types';
import { savedQueryApi } from '$lib/api/client';

function createSavedQueriesStore() {
	const { subscribe, set, update } = writable<SavedQuery[]>([]);

	return {
		subscribe,

		load: async () => {
			try {
				const queries = await savedQueryApi.list();
				set(queries || []);
			} catch (error) {
				console.error('Failed to load saved queries:', error);
				set([]);
			}
		},

		add: async (data: SavedQueryRequest): Promise<SavedQuery> => {
			const query = await savedQueryApi.create(data);
			update((queries) => [...queries, query]);
			return query;
		},

		update: async (id: string, data: SavedQueryRequest): Promise<SavedQuery> => {
			const query = await savedQueryApi.update(id, data);
			update((queries) => queries.map((q) => (q.id === id ? query : q)));
			return query;
		},

		remove: async (id: string): Promise<void> => {
			await savedQueryApi.delete(id);
			update((queries) => queries.filter((q) => q.id !== id));
		}
	};
}

export const savedQueries = createSavedQueriesStore();
