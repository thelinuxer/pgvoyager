import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';
import { themes, themeList, type Theme, type ThemeColors } from './themes';

const STORAGE_KEY = 'pgvoyager-theme';
const DEFAULT_THEME = 'catppuccin-mocha';

function getInitialTheme(): string {
	if (!browser) return DEFAULT_THEME;
	return localStorage.getItem(STORAGE_KEY) || DEFAULT_THEME;
}

function applyTheme(theme: Theme) {
	if (!browser) return;

	const root = document.documentElement;

	// Apply all color variables
	const colorMap: Record<keyof ThemeColors, string> = {
		bg: '--color-bg',
		bgSecondary: '--color-bg-secondary',
		bgTertiary: '--color-bg-tertiary',
		surface: '--color-surface',
		surfaceHover: '--color-surface-hover',
		border: '--color-border',
		text: '--color-text',
		textMuted: '--color-text-muted',
		textDim: '--color-text-dim',
		primary: '--color-primary',
		primaryHover: '--color-primary-hover',
		success: '--color-success',
		warning: '--color-warning',
		error: '--color-error',
		info: '--color-info'
	};

	Object.entries(theme.colors).forEach(([key, value]) => {
		const cssVar = colorMap[key as keyof ThemeColors];
		if (cssVar) {
			root.style.setProperty(cssVar, value);
		}
	});

	// Set data attributes for potential CSS selectors
	root.dataset.theme = theme.id;
	root.dataset.themeType = theme.type;
}

function createThemeStore() {
	const { subscribe, set, update } = writable<string>(getInitialTheme());

	return {
		subscribe,
		setTheme: (themeId: string) => {
			if (!themes[themeId]) return;
			set(themeId);
			if (browser) {
				localStorage.setItem(STORAGE_KEY, themeId);
				applyTheme(themes[themeId]);
			}
		},
		initialize: () => {
			if (browser) {
				const themeId = getInitialTheme();
				if (themes[themeId]) {
					applyTheme(themes[themeId]);
				}
			}
		}
	};
}

export const themeId = createThemeStore();
export const currentTheme = derived(themeId, ($id) => themes[$id] || themes[DEFAULT_THEME]);
export { themes, themeList, type Theme, type ThemeColors };
