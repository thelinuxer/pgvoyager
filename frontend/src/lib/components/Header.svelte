<script lang="ts">
	import { connections, activeConnectionId, activeConnection } from '$lib/stores/connections';
	import { connectionApi } from '$lib/api/client';
	import { layout } from '$lib/stores/layout';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		onNewConnection: () => void;
		onEditConnection: () => void;
		onSettings: () => void;
		onToggleClaude?: () => void;
	}

	let { onNewConnection, onEditConnection, onSettings, onToggleClaude }: Props = $props();

	let isConnecting = $state(false);
	let connectionError = $state<string | null>(null);

	async function handleConnect(id: string) {
		if (isConnecting) return;

		isConnecting = true;
		connectionError = null;

		try {
			await connectionApi.connect(id);
			connections.setConnected(id, true);
			activeConnectionId.set(id);
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Connection failed';
			connectionError = message;
			// Clear error after 5 seconds
			setTimeout(() => {
				connectionError = null;
			}, 5000);
		} finally {
			isConnecting = false;
		}
	}

	async function handleDisconnect() {
		const connId = $activeConnectionId;
		if (!connId) return;

		try {
			await connectionApi.disconnect(connId);
			connections.setConnected(connId, false);
			activeConnectionId.set(null);
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Disconnect failed';
			connectionError = message;
			setTimeout(() => {
				connectionError = null;
			}, 5000);
		}
	}
</script>

<header class="header">
	<div class="header-left">
		<div class="logo">
			<Icon name="layers" size={24} class="logo-icon" />
			<span class="logo-text">PgVoyager</span>
		</div>
	</div>

	<div class="header-center">
		<div class="connection-wrapper" class:connecting={isConnecting} class:error={connectionError}>
			{#if isConnecting}
				<Icon name="refresh" size={14} class="spinning" />
			{:else}
				<Icon name="database" size={14} />
			{/if}
			<select
				class="connection-select"
				value={$activeConnectionId || ''}
				disabled={isConnecting}
				onchange={(e) => {
					const id = e.currentTarget.value;
					if (id) handleConnect(id);
				}}
			>
				<option value="">{isConnecting ? 'Connecting...' : 'Select Connection...'}</option>
				{#each $connections as conn}
					<option value={conn.id}>
						{conn.name} ({conn.host}:{conn.port}/{conn.database})
						{conn.isConnected ? '●' : ''}
					</option>
				{/each}
			</select>
		</div>

		{#if connectionError}
			<div class="connection-error" title={connectionError}>
				<Icon name="alert-circle" size={14} />
				<span class="error-text">{connectionError}</span>
			</div>
		{/if}

		{#if $activeConnection}
			<button class="btn btn-sm btn-ghost" onclick={onEditConnection} title="Edit Connection">
				<Icon name="edit" size={14} />
			</button>
			<button class="btn btn-sm btn-ghost" onclick={handleDisconnect} title="Disconnect">
				<Icon name="power" size={14} />
			</button>
		{/if}
	</div>

	<div class="header-right">
		{#if $activeConnection && onToggleClaude}
			<button
				class="btn btn-sm"
				class:btn-ghost={!$layout.claudeTerminalVisible}
				class:btn-primary={$layout.claudeTerminalVisible}
				onclick={onToggleClaude}
				title="Claude Assistant (Ctrl+`)"
			>
				<Icon name="terminal" size={14} />
				Claude
			</button>
		{/if}
		<a
			href="https://github.com/thelinuxer/pgvoyager"
			target="_blank"
			rel="noopener noreferrer"
			class="btn btn-sm btn-ghost"
			title="View on GitHub"
		>
			<Icon name="github" size={14} />
		</a>
		<button class="btn btn-sm btn-ghost" onclick={onSettings} title="Settings">
			<Icon name="settings" size={14} />
		</button>
		<button class="btn btn-sm btn-ghost" onclick={() => layout.reset()} title="Reset Layout">
			<Icon name="layout" size={14} />
		</button>
		<button class="btn btn-sm btn-primary" onclick={onNewConnection}>
			<Icon name="plus" size={14} />
			New Connection
		</button>
	</div>
</header>

<style>
	.header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 16px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
		height: 48px;
	}

	.header-left,
	.header-right {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.header-center {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.logo {
		display: flex;
		align-items: center;
		gap: 8px;
		font-weight: 600;
	}

	.logo-icon {
		color: var(--color-primary);
	}

	.logo-text {
		font-size: 16px;
	}

	.connection-wrapper {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 0 12px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
	}

	.connection-wrapper svg {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.connection-select {
		min-width: 280px;
		padding: 6px 8px;
		background: transparent;
		border: none;
	}

	.connection-select:focus {
		outline: none;
	}

	.connection-select:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.connection-wrapper.connecting {
		border-color: var(--color-primary);
	}

	.connection-wrapper.error {
		border-color: var(--color-error);
	}

	.connection-error {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		background: rgba(243, 139, 168, 0.15);
		border: 1px solid var(--color-error);
		border-radius: var(--radius-sm);
		color: var(--color-error);
		font-size: 12px;
		max-width: 300px;
	}

	.connection-error svg {
		flex-shrink: 0;
	}

	.error-text {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
