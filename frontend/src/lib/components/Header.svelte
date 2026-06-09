<script lang="ts">
	import { connections, activeConnectionId, activeConnection } from '$lib/stores/connections';
	import { connectionApi, updateApi, type UpdateStatus } from '$lib/api/client';
	import { layout } from '$lib/stores/layout';
	import Icon from '$lib/icons/Icon.svelte';
	import ConnectionDropdown from './ConnectionDropdown.svelte';
	import { onMount } from 'svelte';
	import type { Connection } from '$lib/types';

	interface Props {
		onNewConnection: () => void;
		onEditConnection: (connection: Connection) => void;
		onSettings: () => void;
		onToggleClaude?: () => void;
	}

	let { onNewConnection, onEditConnection, onSettings, onToggleClaude }: Props = $props();

	let isConnecting = $state(false);
	let connectionError = $state<string | null>(null);
	let update = $state<UpdateStatus | null>(null);
	let restarting = $state(false);

	// Polling cadence: the desktop process does the heavy lifting (check +
	// download). The UI only polls to learn when an update is ready.
	const UPDATE_POLL_MS = 30 * 60 * 1000;

	async function refreshUpdateStatus() {
		try {
			update = await updateApi.status();
		} catch {
			// Non-fatal: badge falls back to nothing.
		}
	}

	async function handleRestart() {
		if (restarting) return;
		restarting = true;
		try {
			await updateApi.restart(update?.restartToken);
			// Backend swaps + relaunches; this window will be torn down.
		} catch {
			restarting = false;
		}
	}

	onMount(() => {
		refreshUpdateStatus();
		const timer = setInterval(refreshUpdateStatus, UPDATE_POLL_MS);
		return () => clearInterval(timer);
	});

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
			<img src="/logo.svg" alt="PgVoyager" class="logo-icon" />
			<span class="logo-text">PgVoyager</span>
		</div>
		{#if update}
			{#if restarting}
				<span class="version-badge" title="Updating…">
					<Icon name="refresh" size={12} class="spinning" />
					Updating…
				</span>
			{:else if update.status === 'ready'}
				<button class="version-badge update-ready" onclick={handleRestart}
				        title={update.needsElevation
				          ? `Update ${update.latestVersion} ready — restart to apply (will ask for your admin password)`
				          : `Update ${update.latestVersion} ready — restart to apply`}
				        data-testid="btn-update-restart">
					<span class="update-dot"></span>
					Restart to update
				</button>
			{:else if update.status === 'downloading'}
				<span class="version-badge" title="Downloading update {update.latestVersion}…">
					<Icon name="refresh" size={12} class="spinning" />
					{update.currentVersion}
				</span>
			{:else if update.status === 'manual'}
				<a href={update.releaseUrl} target="_blank" rel="noopener noreferrer"
				   class="version-badge update-available"
				   title="Update available! Click to download {update.latestVersion}">
					<span class="update-dot"></span>
					{update.currentVersion}
				</a>
			{:else}
				<span class="version-badge" title="PgVoyager {update.currentVersion}">
					{update.currentVersion}
				</span>
			{/if}
		{/if}
	</div>

	<div class="header-center">
		<ConnectionDropdown
			{isConnecting}
			onConnect={handleConnect}
			onEdit={onEditConnection}
		/>

		{#if connectionError}
			<div class="connection-error" title={connectionError}>
				<Icon name="alert-circle" size={14} />
				<span class="error-text">{connectionError}</span>
			</div>
		{/if}

		{#if $activeConnection}
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
		height: 28px;
		width: auto;
	}

	.logo-text {
		font-size: 16px;
	}

	.version-badge {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 8px;
		font-size: 11px;
		font-weight: 500;
		color: var(--color-text-muted);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 12px;
	}

	.version-badge.update-available {
		color: var(--color-primary);
		border-color: var(--color-primary);
		background: rgba(137, 180, 250, 0.1);
		text-decoration: none;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.version-badge.update-available:hover {
		background: rgba(137, 180, 250, 0.2);
	}

	.version-badge.update-ready {
		border: none;
		cursor: pointer;
		background: var(--color-primary);
		color: #fff;
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}

	.version-badge.update-ready:hover {
		filter: brightness(1.05);
	}

	.update-dot {
		width: 8px;
		height: 8px;
		background: var(--color-error);
		border-radius: 50%;
		flex-shrink: 0;
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
