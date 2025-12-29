import { writable, derived } from 'svelte/store';
import type { Connection } from '$lib/types';
import { connectionApi } from '$lib/api/client';

function createConnectionsStore() {
	const { subscribe, set, update } = writable<Connection[]>([]);

	return {
		subscribe,
		load: async () => {
			try {
				const connections = await connectionApi.list();
				set(connections);
			} catch (error) {
				console.error('Failed to load connections:', error);
				set([]);
			}
		},
		add: (connection: Connection) => {
			update((conns) => [...conns, connection]);
		},
		remove: (id: string) => {
			update((conns) => conns.filter((c) => c.id !== id));
		},
		updateConnection: (id: string, updates: Partial<Connection>) => {
			update((conns) =>
				conns.map((c) => (c.id === id ? { ...c, ...updates } : c))
			);
		},
		setConnected: (id: string, isConnected: boolean) => {
			update((conns) =>
				conns.map((c) => (c.id === id ? { ...c, isConnected } : c))
			);
		}
	};
}

export const connections = createConnectionsStore();

// Currently active connection
export const activeConnectionId = writable<string | null>(null);

export const activeConnection = derived(
	[connections, activeConnectionId],
	([$connections, $activeConnectionId]) => {
		if (!$activeConnectionId) return null;
		return $connections.find((c) => c.id === $activeConnectionId) || null;
	}
);
