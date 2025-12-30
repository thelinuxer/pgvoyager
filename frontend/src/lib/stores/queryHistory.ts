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

const STORAGE_KEY = 'pgvoyager_query_history';
const MAX_HISTORY_SIZE = 100;

function loadFromStorage(): QueryHistoryEntry[] {
	if (typeof window === 'undefined') return [];
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		return stored ? JSON.parse(stored) : [];
	} catch {
		return [];
	}
}

function saveToStorage(entries: QueryHistoryEntry[]) {
	if (typeof window === 'undefined') return;
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(entries));
	} catch {
		// Storage full or unavailable, silently fail
	}
}

function createQueryHistoryStore() {
	const { subscribe, set, update } = writable<QueryHistoryEntry[]>(loadFromStorage());

	return {
		subscribe,

		add(entry: Omit<QueryHistoryEntry, 'id' | 'executedAt'>) {
			update((entries) => {
				const newEntry: QueryHistoryEntry = {
					...entry,
					id: crypto.randomUUID(),
					executedAt: new Date().toISOString()
				};

				// Add to beginning, limit size
				const updated = [newEntry, ...entries].slice(0, MAX_HISTORY_SIZE);
				saveToStorage(updated);
				return updated;
			});
		},

		remove(id: string) {
			update((entries) => {
				const updated = entries.filter((e) => e.id !== id);
				saveToStorage(updated);
				return updated;
			});
		},

		clear() {
			set([]);
			saveToStorage([]);
		},

		clearForConnection(connectionId: string) {
			update((entries) => {
				const updated = entries.filter((e) => e.connectionId !== connectionId);
				saveToStorage(updated);
				return updated;
			});
		},

		getByConnection(connectionId: string): QueryHistoryEntry[] {
			return get({ subscribe }).filter((e) => e.connectionId === connectionId);
		}
	};
}

export const queryHistory = createQueryHistoryStore();
