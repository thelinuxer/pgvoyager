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
			<span class="logo-icon">🚀</span>
			<span class="logo-text">PgVoyager</span>
		</div>
	</div>

	<div class="header-center">
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

		{#if $activeConnection}
			<button class="btn btn-sm btn-ghost" onclick={handleDisconnect} title="Disconnect">
				⏏
			</button>
		{/if}
	</div>

	<div class="header-right">
		<button class="btn btn-sm btn-secondary" onclick={onNewConnection}>
			+ New Connection
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
		font-size: 20px;
	}

	.logo-text {
		font-size: 16px;
	}

	.connection-select {
		min-width: 300px;
		padding: 6px 12px;
	}
</style>
