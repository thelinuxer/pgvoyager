import { writable, get } from 'svelte/store';

export interface QueryHistoryEntry {
	id: string;
	sql: string;
	connectionId: string;
	connectionName: string;
	executedAt: string;
	duration: number;
	rowCount: number;
	success: boolean;
	error?: string;
}

const API_BASE = '/api/history';

async function loadFromBackend(): Promise<QueryHistoryEntry[]> {
	if (typeof window === 'undefined') return [];
	try {
		const response = await fetch(API_BASE);
		if (!response.ok) return [];
		return await response.json();
	} catch {
		return [];
	}
}

function createQueryHistoryStore() {
	const { subscribe, set, update } = writable<QueryHistoryEntry[]>([]);

	// Load initial data
	if (typeof window !== 'undefined') {
		loadFromBackend().then(set);
	}

	return {
		subscribe,

		async add(entry: Omit<QueryHistoryEntry, 'id' | 'executedAt'>) {
			try {
				const response = await fetch(API_BASE, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(entry)
				});
				if (!response.ok) throw new Error('Failed to add history');

				const newEntry = await response.json();
				update((entries) => [newEntry, ...entries]);
			} catch (e) {
				console.error('Failed to add query history:', e);
			}
		},

		async remove(id: string) {
			try {
				const response = await fetch(`${API_BASE}/${id}`, { method: 'DELETE' });
				if (!response.ok) throw new Error('Failed to remove history');

				update((entries) => entries.filter((e) => e.id !== id));
			} catch (e) {
				console.error('Failed to remove query history:', e);
			}
		},

		async clear() {
			try {
				const response = await fetch(API_BASE, { method: 'DELETE' });
				if (!response.ok) throw new Error('Failed to clear history');

				set([]);
			} catch (e) {
				console.error('Failed to clear query history:', e);
			}
		},

		async clearForConnection(connectionId: string) {
			try {
				const response = await fetch(`${API_BASE}?connectionId=${connectionId}`, {
					method: 'DELETE'
				});
				if (!response.ok) throw new Error('Failed to clear history');

				update((entries) => entries.filter((e) => e.connectionId !== connectionId));
			} catch (e) {
				console.error('Failed to clear connection history:', e);
			}
		},

		async refresh() {
			const entries = await loadFromBackend();
			set(entries);
		},

		getByConnection(connectionId: string): QueryHistoryEntry[] {
			return get({ subscribe }).filter((e) => e.connectionId === connectionId);
		}
	};
}

export const queryHistory = createQueryHistoryStore();
