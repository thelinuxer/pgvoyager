import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export interface LayoutState {
	sidebarWidth: number;
	queryEditorHeight: number; // percentage
	claudeTerminalWidth: number; // pixels - right panel
	claudeTerminalVisible: boolean;
}

const DEFAULT_LAYOUT: LayoutState = {
	sidebarWidth: 280,
	queryEditorHeight: 40, // 40%
	claudeTerminalWidth: 500,
	claudeTerminalVisible: false
};

const PREF_KEY = 'layout';

async function loadLayout(): Promise<LayoutState> {
	if (!browser) return DEFAULT_LAYOUT;

	try {
		const response = await fetch(`/api/preferences/${PREF_KEY}`);
		if (!response.ok) return DEFAULT_LAYOUT;
		const data = await response.json();
		if (data.value) {
			const parsed = JSON.parse(data.value);
			return { ...DEFAULT_LAYOUT, ...parsed };
		}
	} catch (e) {
		console.warn('Failed to load layout from backend:', e);
	}
	return DEFAULT_LAYOUT;
}

async function saveLayout(layout: LayoutState) {
	if (!browser) return;

	try {
		await fetch('/api/preferences', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ key: PREF_KEY, value: JSON.stringify(layout) })
		});
	} catch (e) {
		console.warn('Failed to save layout to backend:', e);
	}
}

function createLayoutStore() {
	const { subscribe, set, update } = writable<LayoutState>(DEFAULT_LAYOUT);

	// Load initial layout
	if (browser) {
		loadLayout().then(set);
	}

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

		setClaudeTerminalWidth: (width: number) => {
			update((state) => {
				const newState = { ...state, claudeTerminalWidth: Math.max(300, Math.min(800, width)) };
				saveLayout(newState);
				return newState;
			});
		},

		setClaudeTerminalVisible: (visible: boolean) => {
			update((state) => {
				const newState = { ...state, claudeTerminalVisible: visible };
				saveLayout(newState);
				return newState;
			});
		},

		toggleClaudeTerminal: () => {
			update((state) => {
				const newState = { ...state, claudeTerminalVisible: !state.claudeTerminalVisible };
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
