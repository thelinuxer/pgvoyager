import { writable, derived } from 'svelte/store';
import type { Tab, TabType } from '$lib/types';

function generateTabId(): string {
	return `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

function createTabsStore() {
	const { subscribe, set, update } = writable<Tab[]>([]);

	return {
		subscribe,

		openTable: (schema: string, table: string) => {
			update((tabs) => {
				// Check if tab already exists
				const existing = tabs.find(
					(t) => t.type === 'table' && t.schema === schema && t.table === table
				);
				if (existing) {
					// Just activate it
					activeTabId.set(existing.id);
					return tabs;
				}

				// Find first unpinned tab to potentially replace
				const unpinnedIndex = tabs.findIndex((t) => !t.isPinned);

				const newTab: Tab = {
					id: generateTabId(),
					type: 'table',
					title: `${schema}.${table}`,
					schema,
					table,
					isPinned: false
				};

				let newTabs: Tab[];
				if (unpinnedIndex === -1) {
					// No unpinned tabs, add new one
					newTabs = [...tabs, newTab];
				} else {
					// Replace the first unpinned tab
					newTabs = [...tabs];
					newTabs[unpinnedIndex] = newTab;
				}

				activeTabId.set(newTab.id);
				return newTabs;
			});
		},

		openQuery: (title?: string) => {
			update((tabs) => {
				const queryCount = tabs.filter((t) => t.type === 'query').length;
				const newTab: Tab = {
					id: generateTabId(),
					type: 'query',
					title: title || `Query ${queryCount + 1}`,
					isPinned: false
				};

				activeTabId.set(newTab.id);
				return [...tabs, newTab];
			});
		},

		openView: (schema: string, view: string) => {
			update((tabs) => {
				const existing = tabs.find(
					(t) => t.type === 'view' && t.schema === schema && t.view === view
				);
				if (existing) {
					activeTabId.set(existing.id);
					return tabs;
				}

				const newTab: Tab = {
					id: generateTabId(),
					type: 'view',
					title: `${schema}.${view}`,
					schema,
					view,
					isPinned: false
				};

				const unpinnedIndex = tabs.findIndex((t) => !t.isPinned);
				let newTabs: Tab[];

				if (unpinnedIndex === -1) {
					newTabs = [...tabs, newTab];
				} else {
					newTabs = [...tabs];
					newTabs[unpinnedIndex] = newTab;
				}

				activeTabId.set(newTab.id);
				return newTabs;
			});
		},

		close: (id: string) => {
			update((tabs) => {
				const index = tabs.findIndex((t) => t.id === id);
				if (index === -1) return tabs;

				const newTabs = tabs.filter((t) => t.id !== id);

				// If closing active tab, activate the nearest tab
				let currentActiveId: string | null = null;
				activeTabId.subscribe((v) => (currentActiveId = v))();

				if (currentActiveId === id && newTabs.length > 0) {
					const newIndex = Math.min(index, newTabs.length - 1);
					activeTabId.set(newTabs[newIndex].id);
				} else if (newTabs.length === 0) {
					activeTabId.set(null);
				}

				return newTabs;
			});
		},

		closeOthers: (id: string) => {
			update((tabs) => {
				const tab = tabs.find((t) => t.id === id);
				if (!tab) return tabs;

				// Keep pinned tabs and the specified tab
				const newTabs = tabs.filter((t) => t.id === id || t.isPinned);
				activeTabId.set(id);
				return newTabs;
			});
		},

		closeAll: () => {
			update((tabs) => {
				// Keep only pinned tabs
				const pinnedTabs = tabs.filter((t) => t.isPinned);
				if (pinnedTabs.length > 0) {
					activeTabId.set(pinnedTabs[0].id);
				} else {
					activeTabId.set(null);
				}
				return pinnedTabs;
			});
		},

		togglePin: (id: string) => {
			update((tabs) =>
				tabs.map((t) => (t.id === id ? { ...t, isPinned: !t.isPinned } : t))
			);
		},

		updateTitle: (id: string, title: string) => {
			update((tabs) =>
				tabs.map((t) => (t.id === id ? { ...t, title } : t))
			);
		},

		setData: (id: string, data: Tab['data']) => {
			update((tabs) =>
				tabs.map((t) => (t.id === id ? { ...t, data } : t))
			);
		},

		reorder: (fromIndex: number, toIndex: number) => {
			update((tabs) => {
				const newTabs = [...tabs];
				const [removed] = newTabs.splice(fromIndex, 1);
				newTabs.splice(toIndex, 0, removed);
				return newTabs;
			});
		}
	};
}

export const tabs = createTabsStore();
export const activeTabId = writable<string | null>(null);

export const activeTab = derived([tabs, activeTabId], ([$tabs, $activeTabId]) => {
	if (!$activeTabId) return null;
	return $tabs.find((t) => t.id === $activeTabId) || null;
});
