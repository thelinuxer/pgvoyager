export interface ThemeColors {
	bg: string;
	bgSecondary: string;
	bgTertiary: string;
	surface: string;
	surfaceHover: string;
	border: string;
	text: string;
	textMuted: string;
	textDim: string;
	primary: string;
	primaryHover: string;
	success: string;
	warning: string;
	error: string;
	info: string;
}

export interface Theme {
	id: string;
	name: string;
	type: 'dark' | 'light';
	colors: ThemeColors;
}

export const themes: Record<string, Theme> = {
	'catppuccin-mocha': {
		id: 'catppuccin-mocha',
		name: 'Catppuccin Mocha',
		type: 'dark',
		colors: {
			bg: '#1e1e2e',
			bgSecondary: '#181825',
			bgTertiary: '#11111b',
			surface: '#313244',
			surfaceHover: '#45475a',
			border: '#45475a',
			text: '#cdd6f4',
			textMuted: '#a6adc8',
			textDim: '#6c7086',
			primary: '#89b4fa',
			primaryHover: '#b4befe',
			success: '#a6e3a1',
			warning: '#f9e2af',
			error: '#f38ba8',
			info: '#89dceb'
		}
	},
	'catppuccin-latte': {
		id: 'catppuccin-latte',
		name: 'Catppuccin Latte',
		type: 'light',
		colors: {
			bg: '#eff1f5',
			bgSecondary: '#e6e9ef',
			bgTertiary: '#dce0e8',
			surface: '#ccd0da',
			surfaceHover: '#bcc0cc',
			border: '#bcc0cc',
			text: '#4c4f69',
			textMuted: '#6c6f85',
			textDim: '#8c8fa1',
			primary: '#1e66f5',
			primaryHover: '#7287fd',
			success: '#40a02b',
			warning: '#df8e1d',
			error: '#d20f39',
			info: '#04a5e5'
		}
	},
	'dracula': {
		id: 'dracula',
		name: 'Dracula',
		type: 'dark',
		colors: {
			bg: '#282a36',
			bgSecondary: '#21222c',
			bgTertiary: '#191a21',
			surface: '#44475a',
			surfaceHover: '#6272a4',
			border: '#44475a',
			text: '#f8f8f2',
			textMuted: '#bfbfbf',
			textDim: '#6272a4',
			primary: '#bd93f9',
			primaryHover: '#ff79c6',
			success: '#50fa7b',
			warning: '#f1fa8c',
			error: '#ff5555',
			info: '#8be9fd'
		}
	},
	'nord': {
		id: 'nord',
		name: 'Nord',
		type: 'dark',
		colors: {
			bg: '#2e3440',
			bgSecondary: '#272c36',
			bgTertiary: '#1f232b',
			surface: '#3b4252',
			surfaceHover: '#434c5e',
			border: '#4c566a',
			text: '#eceff4',
			textMuted: '#d8dee9',
			textDim: '#81a1c1',
			primary: '#88c0d0',
			primaryHover: '#8fbcbb',
			success: '#a3be8c',
			warning: '#ebcb8b',
			error: '#bf616a',
			info: '#81a1c1'
		}
	},
	'solarized-dark': {
		id: 'solarized-dark',
		name: 'Solarized Dark',
		type: 'dark',
		colors: {
			bg: '#002b36',
			bgSecondary: '#073642',
			bgTertiary: '#001f27',
			surface: '#073642',
			surfaceHover: '#094959',
			border: '#586e75',
			text: '#839496',
			textMuted: '#657b83',
			textDim: '#586e75',
			primary: '#268bd2',
			primaryHover: '#2aa198',
			success: '#859900',
			warning: '#b58900',
			error: '#dc322f',
			info: '#2aa198'
		}
	},
	'solarized-light': {
		id: 'solarized-light',
		name: 'Solarized Light',
		type: 'light',
		colors: {
			bg: '#fdf6e3',
			bgSecondary: '#eee8d5',
			bgTertiary: '#ddd6c3',
			surface: '#eee8d5',
			surfaceHover: '#ddd6c3',
			border: '#93a1a1',
			text: '#657b83',
			textMuted: '#839496',
			textDim: '#93a1a1',
			primary: '#268bd2',
			primaryHover: '#2aa198',
			success: '#859900',
			warning: '#b58900',
			error: '#dc322f',
			info: '#2aa198'
		}
	},
	'one-dark': {
		id: 'one-dark',
		name: 'One Dark',
		type: 'dark',
		colors: {
			bg: '#282c34',
			bgSecondary: '#21252b',
			bgTertiary: '#1b1d23',
			surface: '#3e4451',
			surfaceHover: '#4b5263',
			border: '#3e4451',
			text: '#abb2bf',
			textMuted: '#8b929e',
			textDim: '#5c6370',
			primary: '#61afef',
			primaryHover: '#528bce',
			success: '#98c379',
			warning: '#e5c07b',
			error: '#e06c75',
			info: '#56b6c2'
		}
	},
	'github-dark': {
		id: 'github-dark',
		name: 'GitHub Dark',
		type: 'dark',
		colors: {
			bg: '#0d1117',
			bgSecondary: '#161b22',
			bgTertiary: '#010409',
			surface: '#21262d',
			surfaceHover: '#30363d',
			border: '#30363d',
			text: '#c9d1d9',
			textMuted: '#8b949e',
			textDim: '#6e7681',
			primary: '#58a6ff',
			primaryHover: '#79c0ff',
			success: '#3fb950',
			warning: '#d29922',
			error: '#f85149',
			info: '#39c5cf'
		}
	},
	'github-light': {
		id: 'github-light',
		name: 'GitHub Light',
		type: 'light',
		colors: {
			bg: '#ffffff',
			bgSecondary: '#f6f8fa',
			bgTertiary: '#eaeef2',
			surface: '#f6f8fa',
			surfaceHover: '#eaeef2',
			border: '#d0d7de',
			text: '#24292f',
			textMuted: '#57606a',
			textDim: '#8c959f',
			primary: '#0969da',
			primaryHover: '#0550ae',
			success: '#1a7f37',
			warning: '#9a6700',
			error: '#cf222e',
			info: '#0969da'
		}
	}
};

export const themeList = Object.values(themes);
