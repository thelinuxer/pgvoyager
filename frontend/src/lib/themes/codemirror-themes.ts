import { oneDark } from '@codemirror/theme-one-dark';
import type { Extension } from '@codemirror/state';

// Map app themes to CodeMirror themes
// For themes without exact CodeMirror matches, use closest alternatives
export function getCodeMirrorTheme(themeId: string): Extension[] {
	switch (themeId) {
		// Dark themes use oneDark
		case 'catppuccin-mocha':
		case 'dracula':
		case 'one-dark':
		case 'github-dark':
		case 'nord':
		case 'solarized-dark':
			return [oneDark];

		// Light themes use default CodeMirror styling
		case 'catppuccin-latte':
		case 'solarized-light':
		case 'github-light':
			return [];

		default:
			return [oneDark];
	}
}
