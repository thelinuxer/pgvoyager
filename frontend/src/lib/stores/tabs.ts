import { writable, derived, get } from 'svelte/store';
import type { Tab, TableLocation, ERDLocation } from '$lib/types';

function generateTabId(): string {
	return `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

function createTabsStore() {
	const { subscribe, set, update } = writable<Tab[]>([]);

	return {
		subscribe,

		openTable: (
			schema: string,
			table: string,
			options?: {
				filter?: { column: string; value: string };
				sort?: { column: string; direction: 'ASC' | 'DESC' };
				limit?: number;
				forceNew?: boolean;
			}
		) => {
			update((tabs) => {
				// Check if tab already exists (only for requests without special options)
				const existing = tabs.find(
					(t) => t.type === 'table' && t.schema === schema && t.table === table
				);
				if (existing && !options?.filter && !options?.sort && !options?.forceNew) {
					// Just activate it (only for simple requests)
					activeTabId.set(existing.id);
					return tabs;
				}

				// Find first unpinned tab to potentially replace
				const unpinnedIndex = tabs.findIndex((t) => !t.isPinned);

				const initialLocation: TableLocation = {
					schema,
					table,
					filter: options?.filter,
					sort: options?.sort,
					limit: options?.limit
				};

				let titleSuffix = '';
				if (options?.filter) titleSuffix = ' (filtered)';
				else if (options?.sort?.direction === 'DESC') titleSuffix = ' (last)';
				else if (options?.sort) titleSuffix = ' (first)';

				const newTab: Tab = {
					id: generateTabId(),
					type: 'table',
					title: `${schema}.${table}${titleSuffix}`,
					schema,
					table,
					isPinned: false,
					navigationStack: [initialLocation],
					navigationIndex: 0
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

		openQuery: (options?: { title?: string; initialSql?: string }) => {
			update((tabs) => {
				const queryCount = tabs.filter((t) => t.type === 'query').length;
				const newTab: Tab = {
					id: generateTabId(),
					type: 'query',
					title: options?.title || `Query ${queryCount + 1}`,
					isPinned: false,
					initialSql: options?.initialSql
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

		openFunction: (schema: string, functionName: string) => {
			update((tabs) => {
				const existing = tabs.find(
					(t) => t.type === 'function' && t.schema === schema && t.functionName === functionName
				);
				if (existing) {
					activeTabId.set(existing.id);
					return tabs;
				}

				const newTab: Tab = {
					id: generateTabId(),
					type: 'function',
					title: `${schema}.${functionName}`,
					schema,
					functionName,
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

		openSequence: (schema: string, sequenceName: string) => {
			update((tabs) => {
				const existing = tabs.find(
					(t) => t.type === 'sequence' && t.schema === schema && t.sequenceName === sequenceName
				);
				if (existing) {
					activeTabId.set(existing.id);
					return tabs;
				}

				const newTab: Tab = {
					id: generateTabId(),
					type: 'sequence',
					title: `${schema}.${sequenceName}`,
					schema,
					sequenceName,
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

		openType: (schema: string, typeName: string) => {
			update((tabs) => {
				const existing = tabs.find(
					(t) => t.type === 'type' && t.schema === schema && t.typeName === typeName
				);
				if (existing) {
					activeTabId.set(existing.id);
					return tabs;
				}

				const newTab: Tab = {
					id: generateTabId(),
					type: 'type',
					title: `${schema}.${typeName}`,
					schema,
					typeName,
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

		// Open ERD for a specific table (table-centered view)
		openTableERD: (schema: string, table: string) => {
			update((tabs) => {
				const initialLocation: ERDLocation = { schema, centeredTable: table };
				const newTab: Tab = {
					id: generateTabId(),
					type: 'erd',
					title: `ERD: ${schema}.${table}`,
					schema,
					table,
					isPinned: false,
					erdNavigationStack: [initialLocation],
					erdNavigationIndex: 0
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

		// Open ERD for entire schema (full schema view)
		openSchemaERD: (schema: string) => {
			update((tabs) => {
				const initialLocation: ERDLocation = { schema };
				const newTab: Tab = {
					id: generateTabId(),
					type: 'erd',
					title: `ERD: ${schema}`,
					schema,
					isPinned: false,
					erdNavigationStack: [initialLocation],
					erdNavigationIndex: 0
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

		// Navigate to a different table within ERD tab (push to stack)
		navigateERD: (tabId: string, schema: string, table?: string) => {
			update((tabs) => {
				return tabs.map((t) => {
					if (t.id !== tabId || t.type !== 'erd') return t;

					const stack = t.erdNavigationStack || [];
					const index = t.erdNavigationIndex ?? 0;

					// Truncate forward history and add new location
					const newStack = stack.slice(0, index + 1);
					const newLocation: ERDLocation = { schema, centeredTable: table };
					newStack.push(newLocation);

					// Limit stack to 50 entries
					if (newStack.length > 50) {
						newStack.shift();
					}

					return {
						...t,
						schema,
						table,
						title: table ? `ERD: ${schema}.${table}` : `ERD: ${schema}`,
						erdNavigationStack: newStack,
						erdNavigationIndex: newStack.length - 1
					};
				});
			});
		},

		// Go back in ERD navigation stack
		navigateERDBack: (tabId: string): ERDLocation | null => {
			let targetLocation: ERDLocation | null = null;

			update((tabs) => {
				return tabs.map((t) => {
					if (t.id !== tabId || t.type !== 'erd') return t;

					const stack = t.erdNavigationStack || [];
					const index = t.erdNavigationIndex ?? 0;

					if (index > 0) {
						const newIndex = index - 1;
						targetLocation = stack[newIndex];
						return {
							...t,
							schema: targetLocation.schema,
							table: targetLocation.centeredTable,
							title: targetLocation.centeredTable
								? `ERD: ${targetLocation.schema}.${targetLocation.centeredTable}`
								: `ERD: ${targetLocation.schema}`,
							erdNavigationIndex: newIndex
						};
					}
					return t;
				});
			});

			return targetLocation;
		},

		// Go forward in ERD navigation stack
		navigateERDForward: (tabId: string): ERDLocation | null => {
			let targetLocation: ERDLocation | null = null;

			update((tabs) => {
				return tabs.map((t) => {
					if (t.id !== tabId || t.type !== 'erd') return t;

					const stack = t.erdNavigationStack || [];
					const index = t.erdNavigationIndex ?? 0;

					if (index < stack.length - 1) {
						const newIndex = index + 1;
						targetLocation = stack[newIndex];
						return {
							...t,
							schema: targetLocation.schema,
							table: targetLocation.centeredTable,
							title: targetLocation.centeredTable
								? `ERD: ${targetLocation.schema}.${targetLocation.centeredTable}`
								: `ERD: ${targetLocation.schema}`,
							erdNavigationIndex: newIndex
						};
					}
					return t;
				});
			});

			return targetLocation;
		},

		// Check if can go back in ERD navigation
		canNavigateERDBack: (tabId: string): boolean => {
			const allTabs = get({ subscribe });
			const tab = allTabs.find((t) => t.id === tabId);
			if (!tab || tab.type !== 'erd') return false;
			return (tab.erdNavigationIndex ?? 0) > 0;
		},

		// Check if can go forward in ERD navigation
		canNavigateERDForward: (tabId: string): boolean => {
			const allTabs = get({ subscribe });
			const tab = allTabs.find((t) => t.id === tabId);
			if (!tab || tab.type !== 'erd') return false;
			const stack = tab.erdNavigationStack || [];
			return (tab.erdNavigationIndex ?? 0) < stack.length - 1;
		},

		// Get current ERD location for a tab
		getCurrentERDLocation: (tabId: string): ERDLocation | null => {
			const allTabs = get({ subscribe });
			const tab = allTabs.find((t) => t.id === tabId);
			if (!tab || tab.type !== 'erd') return null;
			const stack = tab.erdNavigationStack || [];
			const index = tab.erdNavigationIndex ?? 0;
			return stack[index] || null;
		},

		// Handle FK click - respects tab pinning behavior
		// If current tab is pinned: open new tab with filter
		// If current tab is unpinned: navigate within the tab
		handleFKClick: (tabId: string, schema: string, table: string, filterColumn: string, filterValue: string) => {
			const allTabs = get({ subscribe });
			const currentTab = allTabs.find((t) => t.id === tabId);

			if (!currentTab) return;

			const filter = { column: filterColumn, value: filterValue };

			if (currentTab.isPinned) {
				// Pinned tab: open in new tab (or replace unpinned tab)
				// Use a custom update to handle this
				update((tabs) => {
					// Find first unpinned tab to potentially replace
					const unpinnedIndex = tabs.findIndex((t) => !t.isPinned);

					const initialLocation: TableLocation = { schema, table, filter };
					const newTab: Tab = {
						id: generateTabId(),
						type: 'table',
						title: `${schema}.${table}`,
						schema,
						table,
						isPinned: false,
						navigationStack: [initialLocation],
						navigationIndex: 0
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
			} else {
				// Unpinned tab: navigate within the same tab
				update((tabs) => {
					return tabs.map((t) => {
						if (t.id !== tabId || t.type !== 'table') return t;

						const stack = t.navigationStack || [];
						const index = t.navigationIndex ?? 0;

						// Truncate forward history and add new location
						const newStack = stack.slice(0, index + 1);
						const newLocation: TableLocation = { schema, table, filter };
						newStack.push(newLocation);

						// Limit stack to 50 entries
						if (newStack.length > 50) {
							newStack.shift();
						}

						return {
							...t,
							schema,
							table,
							title: `${schema}.${table}`,
							navigationStack: newStack,
							navigationIndex: newStack.length - 1,
							data: undefined
						};
					});
				});
			}
		},

		// Navigate to a foreign key reference within the current tab (internal use)
		navigateToFK: (tabId: string, schema: string, table: string, filterColumn?: string, filterValue?: string) => {
			update((tabs) => {
				return tabs.map((t) => {
					if (t.id !== tabId || t.type !== 'table') return t;

					const stack = t.navigationStack || [];
					const index = t.navigationIndex ?? 0;

					// Truncate forward history and add new location
					const newStack = stack.slice(0, index + 1);
					const newLocation: TableLocation = {
						schema,
						table,
						filter: filterColumn && filterValue ? { column: filterColumn, value: filterValue } : undefined
					};
					newStack.push(newLocation);

					// Limit stack to 50 entries
					if (newStack.length > 50) {
						newStack.shift();
					}

					return {
						...t,
						schema,
						table,
						title: `${schema}.${table}`,
						navigationStack: newStack,
						navigationIndex: newStack.length - 1,
						data: undefined // Clear data to trigger reload
					};
				});
			});
		},

		// Go back in navigation stack
		navigateBack: (tabId: string): TableLocation | null => {
			let targetLocation: TableLocation | null = null;

			update((tabs) => {
				return tabs.map((t) => {
					if (t.id !== tabId || t.type !== 'table') return t;

					const stack = t.navigationStack || [];
					const index = t.navigationIndex ?? 0;

					if (index > 0) {
						const newIndex = index - 1;
						targetLocation = stack[newIndex];
						return {
							...t,
							schema: targetLocation.schema,
							table: targetLocation.table,
							title: `${targetLocation.schema}.${targetLocation.table}`,
							navigationIndex: newIndex,
							data: undefined
						};
					}
					return t;
				});
			});

			return targetLocation;
		},

		// Go forward in navigation stack
		navigateForward: (tabId: string): TableLocation | null => {
			let targetLocation: TableLocation | null = null;

			update((tabs) => {
				return tabs.map((t) => {
					if (t.id !== tabId || t.type !== 'table') return t;

					const stack = t.navigationStack || [];
					const index = t.navigationIndex ?? 0;

					if (index < stack.length - 1) {
						const newIndex = index + 1;
						targetLocation = stack[newIndex];
						return {
							...t,
							schema: targetLocation.schema,
							table: targetLocation.table,
							title: `${targetLocation.schema}.${targetLocation.table}`,
							navigationIndex: newIndex,
							data: undefined
						};
					}
					return t;
				});
			});

			return targetLocation;
		},

		// Check if can go back/forward
		canNavigateBack: (tabId: string): boolean => {
			const allTabs = get({ subscribe });
			const tab = allTabs.find((t) => t.id === tabId);
			if (!tab || tab.type !== 'table') return false;
			return (tab.navigationIndex ?? 0) > 0;
		},

		canNavigateForward: (tabId: string): boolean => {
			const allTabs = get({ subscribe });
			const tab = allTabs.find((t) => t.id === tabId);
			if (!tab || tab.type !== 'table') return false;
			const stack = tab.navigationStack || [];
			return (tab.navigationIndex ?? 0) < stack.length - 1;
		},

		// Get current location for a tab
		getCurrentLocation: (tabId: string): TableLocation | null => {
			const allTabs = get({ subscribe });
			const tab = allTabs.find((t) => t.id === tabId);
			if (!tab || tab.type !== 'table') return null;
			const stack = tab.navigationStack || [];
			const index = tab.navigationIndex ?? 0;
			return stack[index] || null;
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
