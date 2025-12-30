<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { claudeTerminal } from '$lib/stores/claudeTerminal';
	import { activeConnectionId } from '$lib/stores/connections';
	import { layout } from '$lib/stores/layout';
	import { editorStore } from '$lib/stores/editor';
	import ResizeHandle from './ResizeHandle.svelte';
	import Icon from '$lib/icons/Icon.svelte';

	// Dynamic import for xterm to handle CommonJS module
	type TerminalType = import('@xterm/xterm').Terminal;
	type FitAddonType = import('@xterm/addon-fit').FitAddon;
	let TerminalClass: typeof import('@xterm/xterm').Terminal;
	let FitAddonClass: typeof import('@xterm/addon-fit').FitAddon;

	let terminalContainer: HTMLDivElement;
	let terminal: TerminalType | null = null;
	let fitAddon: FitAddonType | null = null;
	let isInitializing = $state(false);
	let initError = $state<string | null>(null);

	onMount(async () => {
		// Dynamically import xterm modules (CommonJS compatibility)
		const xtermModule = await import('@xterm/xterm');
		const fitAddonModule = await import('@xterm/addon-fit');
		TerminalClass = xtermModule.Terminal;
		FitAddonClass = fitAddonModule.FitAddon;

		// Import CSS
		await import('@xterm/xterm/css/xterm.css');

		await initTerminal();
	});

	onDestroy(() => {
		claudeTerminal.destroySession();
		terminal?.dispose();
	});

	async function initTerminal() {
		if (!$activeConnectionId) {
			initError = 'No active database connection';
			return;
		}

		if (!TerminalClass || !FitAddonClass) {
			initError = 'Terminal modules not loaded';
			return;
		}

		isInitializing = true;
		initError = null;

		try {
			// Create terminal with proper settings for animations/spinners
			terminal = new TerminalClass({
				cursorBlink: true,
				fontSize: 13,
				fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, monospace',
				scrollback: 5000,
				convertEol: false, // Don't convert - preserve escape sequences
				allowProposedApi: true,
				theme: {
					background: '#1e1e2e',
					foreground: '#cdd6f4',
					cursor: '#f5e0dc',
					cursorAccent: '#1e1e2e',
					selectionBackground: '#585b70',
					black: '#45475a',
					red: '#f38ba8',
					green: '#a6e3a1',
					yellow: '#f9e2af',
					blue: '#89b4fa',
					magenta: '#f5c2e7',
					cyan: '#94e2d5',
					white: '#bac2de',
					brightBlack: '#585b70',
					brightRed: '#f38ba8',
					brightGreen: '#a6e3a1',
					brightYellow: '#f9e2af',
					brightBlue: '#89b4fa',
					brightMagenta: '#f5c2e7',
					brightCyan: '#94e2d5',
					brightWhite: '#a6adc8'
				}
			});

			fitAddon = new FitAddonClass();
			terminal.loadAddon(fitAddon);
			terminal.open(terminalContainer);
			fitAddon.fit();

			// Create Claude session
			const sessionId = await claudeTerminal.createSession($activeConnectionId);
			if (!sessionId) {
				throw new Error('Failed to create Claude session');
			}

			// Set initial size before connecting (will be sent when WebSocket opens)
			claudeTerminal.resize(terminal.cols, terminal.rows);

			// Connect terminal to WebSocket
			claudeTerminal.connect(terminal);

			// Refit after connection is established
			setTimeout(() => {
				if (fitAddon && terminal) {
					fitAddon.fit();
					claudeTerminal.resize(terminal.cols, terminal.rows);
				}
			}, 100);
		} catch (e) {
			initError = e instanceof Error ? e.message : 'Failed to initialize terminal';
		} finally {
			isInitializing = false;
		}
	}

	// Handle resize from drag (horizontal - width)
	function handleResize(delta: number) {
		const newWidth = $layout.claudeTerminalWidth - delta;
		layout.setClaudeTerminalWidth(newWidth);

		// Refit terminal after resize
		setTimeout(() => {
			if (fitAddon && terminal) {
				fitAddon.fit();
				claudeTerminal.resize(terminal.cols, terminal.rows);
			}
		}, 50);
	}

	// Sync editor content to Claude session
	$effect(() => {
		const content = $editorStore.content;
		// Only send updates when user changes editor (not when Claude changes it)
		if ($editorStore.lastUpdatedBy === 'user' && $claudeTerminal.isConnected) {
			claudeTerminal.updateEditorState({ content });
		}
	});

	function handleClose() {
		layout.setClaudeTerminalVisible(false);
	}

	// Handle window resize
	function handleWindowResize() {
		if (fitAddon && terminal) {
			fitAddon.fit();
			claudeTerminal.resize(terminal.cols, terminal.rows);
		}
	}

	$effect(() => {
		if (typeof window !== 'undefined') {
			window.addEventListener('resize', handleWindowResize);
			return () => window.removeEventListener('resize', handleWindowResize);
		}
	});

	// Watch for connection changes and update session connection (no restart needed)
	let previousConnectionId: string | null = null;
	$effect(() => {
		const currentConnectionId = $activeConnectionId;

		// Skip if no connection or terminal not ready
		if (!currentConnectionId || !terminal || !TerminalClass || !FitAddonClass) {
			previousConnectionId = currentConnectionId;
			return;
		}

		// Check if connection changed (and we had a previous connection)
		if (previousConnectionId && previousConnectionId !== currentConnectionId) {
			// Connection changed - update session connection (MCP will use new DB)
			handleConnectionChange(currentConnectionId);
		}

		previousConnectionId = currentConnectionId;
	});

	async function handleConnectionChange(newConnectionId: string) {
		// Simply update the session's connection - no restart needed
		// The MCP server calls the backend API which looks up the current connection
		const success = await claudeTerminal.ensureSessionForConnection(newConnectionId);
		if (!success) {
			initError = 'Failed to switch database connection';
		}
	}
</script>

<div class="claude-terminal-panel" style="width: {$layout.claudeTerminalWidth}px">
	<ResizeHandle direction="horizontal" onResize={handleResize} />

	<div class="panel-content">
		<div class="panel-header">
			<div class="panel-title">
				<Icon name="terminal" size={16} />
				<span>Claude Assistant</span>
			</div>
			<div class="panel-actions">
				{#if $claudeTerminal.isConnected}
					<span class="connection-status connected">Connected</span>
				{:else if $claudeTerminal.isConnecting}
					<span class="connection-status connecting">Connecting...</span>
				{:else}
					<span class="connection-status disconnected">Disconnected</span>
				{/if}
				<button class="btn btn-ghost btn-sm" onclick={handleClose} title="Close (Ctrl+`)">
					<Icon name="x" size={16} />
				</button>
			</div>
		</div>

		<div class="terminal-container" bind:this={terminalContainer}>
			{#if isInitializing}
				<div class="terminal-status">
					<Icon name="refresh" size={24} class="spinning" />
					<span>Starting Claude Code...</span>
				</div>
			{:else if initError}
				<div class="terminal-error">
					<Icon name="alert-triangle" size={24} />
					<span>{initError}</span>
					<button class="btn btn-primary btn-sm" onclick={initTerminal}>Retry</button>
				</div>
			{:else if !$activeConnectionId}
				<div class="terminal-status">
					<Icon name="database" size={24} />
					<span>Connect to a database to use Claude Assistant</span>
				</div>
			{/if}
		</div>
	</div>
</div>

<style>
	.claude-terminal-panel {
		display: flex;
		flex-direction: row;
		background: #1e1e2e;
		border-left: 1px solid var(--color-border);
		position: relative;
		height: 100%;
	}

	.panel-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
	}

	.panel-title {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text);
	}

	.panel-actions {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.connection-status {
		font-size: 10px;
		font-weight: normal;
		padding: 2px 6px;
		border-radius: 4px;
	}

	.connection-status.connected {
		background: rgba(166, 227, 161, 0.2);
		color: #a6e3a1;
	}

	.connection-status.connecting {
		background: rgba(249, 226, 175, 0.2);
		color: #f9e2af;
	}

	.connection-status.disconnected {
		background: rgba(243, 139, 168, 0.2);
		color: #f38ba8;
	}

	.terminal-container {
		flex: 1;
		overflow: hidden;
		padding: 8px;
	}

	.terminal-container :global(.xterm) {
		height: 100%;
	}

	.terminal-container :global(.xterm-viewport) {
		overflow-y: auto !important;
	}

	.terminal-status,
	.terminal-error {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 12px;
		color: var(--color-text-muted);
		text-align: center;
		padding: 20px;
	}

	.terminal-error {
		color: var(--color-error);
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}
</style>
