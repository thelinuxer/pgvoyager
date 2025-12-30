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

const STORAGE_KEY = 'pgvoyager-icon-library';
const DEFAULT_LIBRARY: IconLibrary = 'lucide';

function getInitialLibrary(): IconLibrary {
	if (!browser) return DEFAULT_LIBRARY;
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored && ['lucide', 'heroicons', 'phosphor', 'tabler'].includes(stored)) {
		return stored as IconLibrary;
	}
	return DEFAULT_LIBRARY;
}

function createIconLibraryStore() {
	const { subscribe, set } = writable<IconLibrary>(getInitialLibrary());

	return {
		subscribe,
		setLibrary: (library: IconLibrary) => {
			set(library);
			if (browser) {
				localStorage.setItem(STORAGE_KEY, library);
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
