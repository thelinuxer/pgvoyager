import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

export type IconLibrary = 'lucide' | 'heroicons' | 'phosphor' | 'tabler';

export interface IconData {
	viewBox: string;
	paths: Array<{
		d: string;
		fill?: string;
		stroke?: string;
		strokeWidth?: number;
	}>;
}

const PREF_KEY = 'icon-library';
const DEFAULT_LIBRARY: IconLibrary = 'lucide';

async function getInitialLibrary(): Promise<IconLibrary> {
	if (!browser) return DEFAULT_LIBRARY;
	try {
		const response = await fetch(`/api/preferences/${PREF_KEY}`);
		if (!response.ok) return DEFAULT_LIBRARY;
		const data = await response.json();
		const stored = data.value;
		if (stored && ['lucide', 'heroicons', 'phosphor', 'tabler'].includes(stored)) {
			return stored as IconLibrary;
		}
	} catch {
		// Fall through to default
	}
	return DEFAULT_LIBRARY;
}

function createIconLibraryStore() {
	const { subscribe, set } = writable<IconLibrary>(DEFAULT_LIBRARY);

	// Load initial library
	if (browser) {
		getInitialLibrary().then(set);
	}

	return {
		subscribe,
		setLibrary: async (library: IconLibrary) => {
			set(library);
			if (browser) {
				try {
					await fetch('/api/preferences', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ key: PREF_KEY, value: library })
					});
				} catch (e) {
					console.error('Failed to save icon library preference:', e);
				}
			}
		},
		get: () => get({ subscribe })
	};
}

export const iconLibrary = createIconLibraryStore();

export const iconLibraries: { id: IconLibrary; name: string }[] = [
	{ id: 'lucide', name: 'Lucide' },
	{ id: 'heroicons', name: 'Heroicons' },
	{ id: 'phosphor', name: 'Phosphor' },
	{ id: 'tabler', name: 'Tabler Icons' }
];
