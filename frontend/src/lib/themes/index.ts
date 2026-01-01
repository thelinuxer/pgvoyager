import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';
import { themes, themeList, type Theme, type ThemeColors } from './themes';

const DEFAULT_THEME = 'catppuccin-mocha';
const PREF_KEY = 'theme';

async function getInitialTheme(): Promise<string> {
	if (!browser) return DEFAULT_THEME;
	try {
		const response = await fetch(`/api/preferences/${PREF_KEY}`);
		if (!response.ok) return DEFAULT_THEME;
		const data = await response.json();
		return data.value || DEFAULT_THEME;
	} catch {
		return DEFAULT_THEME;
	}
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
	const { subscribe, set } = writable<string>(DEFAULT_THEME);

	return {
		subscribe,
		setTheme: async (themeId: string) => {
			if (!themes[themeId]) return;
			set(themeId);
			if (browser) {
				applyTheme(themes[themeId]);
				// Save to backend
				try {
					await fetch('/api/preferences', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ key: PREF_KEY, value: themeId })
					});
				} catch (e) {
					console.error('Failed to save theme preference:', e);
				}
			}
		},
		initialize: async () => {
			if (browser) {
				const themeId = await getInitialTheme();
				set(themeId);
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
