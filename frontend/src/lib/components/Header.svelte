<script lang="ts">
	import { connections, activeConnectionId, activeConnection } from '$lib/stores/connections';
	import { connectionApi } from '$lib/api/client';

	interface Props {
		onNewConnection: () => void;
	}

	let { onNewConnection }: Props = $props();

	async function handleConnect(id: string) {
		try {
			await connectionApi.connect(id);
			connections.setConnected(id, true);
			activeConnectionId.set(id);
		} catch (error) {
			alert(`Failed to connect: ${error}`);
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
			alert(`Failed to disconnect: ${error}`);
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
		<div class="connection-wrapper">
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
				<path d="M22 6l-10 7L2 6"/>
			</svg>
			<select
				class="connection-select"
				value={$activeConnectionId || ''}
				onchange={(e) => {
					const id = e.currentTarget.value;
					if (id) handleConnect(id);
				}}
			>
				<option value="">Select Connection...</option>
				{#each $connections as conn}
					<option value={conn.id}>
						{conn.name} ({conn.host}:{conn.port}/{conn.database})
						{conn.isConnected ? '●' : ''}
					</option>
				{/each}
			</select>
		</div>

		{#if $activeConnection}
			<button class="btn btn-sm btn-ghost" onclick={handleDisconnect} title="Disconnect">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M18.36 6.64a9 9 0 11-12.73 0"/>
					<line x1="12" y1="2" x2="12" y2="12"/>
				</svg>
			</button>
		{/if}
	</div>

	<div class="header-right">
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
</style>
