import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export interface LayoutState {
	sidebarWidth: number;
	queryEditorHeight: number; // percentage
}

const DEFAULT_LAYOUT: LayoutState = {
	sidebarWidth: 280,
	queryEditorHeight: 40 // 40%
};

const STORAGE_KEY = 'pgvoyager-layout';

function loadLayout(): LayoutState {
	if (!browser) return DEFAULT_LAYOUT;

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			const parsed = JSON.parse(stored);
			return { ...DEFAULT_LAYOUT, ...parsed };
		}
	} catch (e) {
		console.warn('Failed to load layout from localStorage:', e);
	}
	return DEFAULT_LAYOUT;
}

function saveLayout(layout: LayoutState) {
	if (!browser) return;

	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(layout));
	} catch (e) {
		console.warn('Failed to save layout to localStorage:', e);
	}
}

function createLayoutStore() {
	const { subscribe, set, update } = writable<LayoutState>(loadLayout());

	return {
		subscribe,

		setSidebarWidth: (width: number) => {
			update((state) => {
				const newState = { ...state, sidebarWidth: Math.max(200, Math.min(600, width)) };
				saveLayout(newState);
				return newState;
			});
		},

		setQueryEditorHeight: (height: number) => {
			update((state) => {
				const newState = { ...state, queryEditorHeight: Math.max(20, Math.min(80, height)) };
				saveLayout(newState);
				return newState;
			});
		},

		reset: () => {
			set(DEFAULT_LAYOUT);
			saveLayout(DEFAULT_LAYOUT);
		}
	};
}

export const layout = createLayoutStore();
