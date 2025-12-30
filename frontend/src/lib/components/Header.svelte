<script lang="ts">
	import { connections, activeConnectionId, activeConnection } from '$lib/stores/connections';
	import { connectionApi } from '$lib/api/client';
	import { layout } from '$lib/stores/layout';

	interface Props {
		onNewConnection: () => void;
		onEditConnection: () => void;
	}

	let { onNewConnection, onEditConnection }: Props = $props();

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
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="logo-icon">
				<path d="M12 2L2 7l10 5 10-5-10-5z"/>
				<path d="M2 17l10 5 10-5"/>
				<path d="M2 12l10 5 10-5"/>
			</svg>
			<span class="logo-text">PgVoyager</span>
		</div>
	</div>

	<div class="header-center">
		<div class="connection-wrapper" class:connecting={isConnecting} class:error={connectionError}>
			{#if isConnecting}
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spinning">
					<path d="M23 4v6h-6M1 20v-6h6"/>
					<path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
				</svg>
			{:else}
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
					<path d="M22 6l-10 7L2 6"/>
				</svg>
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
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10"/>
					<path d="M12 8v4M12 16h.01"/>
				</svg>
				<span class="error-text">{connectionError}</span>
			</div>
		{/if}

		{#if $activeConnection}
			<button class="btn btn-sm btn-ghost" onclick={onEditConnection} title="Edit Connection">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
					<path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
				</svg>
			</button>
			<button class="btn btn-sm btn-ghost" onclick={handleDisconnect} title="Disconnect">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M18.36 6.64a9 9 0 11-12.73 0"/>
					<line x1="12" y1="2" x2="12" y2="12"/>
				</svg>
			</button>
		{/if}
	</div>

	<div class="header-right">
		<button class="btn btn-sm btn-ghost" onclick={() => layout.reset()} title="Reset Layout">
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
				<line x1="3" y1="9" x2="21" y2="9"/>
				<line x1="9" y1="21" x2="9" y2="9"/>
			</svg>
		</button>
		<button class="btn btn-sm btn-primary" onclick={onNewConnection}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="12" y1="5" x2="12" y2="19"/>
				<line x1="5" y1="12" x2="19" y2="12"/>
			</svg>
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
