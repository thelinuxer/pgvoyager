<script lang="ts">
	import { activeConnection, activeConnectionId, connections } from '$lib/stores/connections';
	import { clearSchema, refreshSchema } from '$lib/stores/schema';
	import { connectionApi, schemaApi } from '$lib/api/client';
	import type { Database } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	let isOpen = $state(false);
	let databases = $state<Database[]>([]);
	let isLoading = $state(false);
	let isSwitching = $state(false);
	let error = $state<string | null>(null);
	let dropdownRef: HTMLDivElement | null = $state(null);

	let loadedForConnId = $state<string | null>(null);

	$effect(() => {
		const connId = $activeConnectionId;
		if (!connId) {
			databases = [];
			loadedForConnId = null;
			return;
		}
		if (connId !== loadedForConnId) {
			databases = [];
			loadedForConnId = connId;
		}
	});

	async function loadDatabases() {
		if (!$activeConnectionId) return;
		isLoading = true;
		error = null;
		try {
			const list = await schemaApi.listDatabases($activeConnectionId);
			databases = list || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load databases';
		} finally {
			isLoading = false;
		}
	}

	async function handleToggle() {
		if (!$activeConnection || isSwitching) return;
		isOpen = !isOpen;
		if (isOpen && databases.length === 0 && !isLoading) {
			await loadDatabases();
		}
	}

	async function handleSelect(dbName: string) {
		if (!$activeConnectionId || !$activeConnection) return;
		if (dbName === $activeConnection.database) {
			isOpen = false;
			return;
		}

		isSwitching = true;
		error = null;
		try {
			const updated = await connectionApi.switchDatabase($activeConnectionId, dbName);
			connections.updateConnection(updated.id, updated);
			clearSchema();
			refreshSchema();
			isOpen = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to switch database';
		} finally {
			isSwitching = false;
		}
	}

	function handleClickOutside(e: MouseEvent) {
		if (dropdownRef && !dropdownRef.contains(e.target as Node)) {
			isOpen = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') isOpen = false;
	}

	$effect(() => {
		if (isOpen) {
			document.addEventListener('click', handleClickOutside);
			document.addEventListener('keydown', handleKeydown);
		}
		return () => {
			document.removeEventListener('click', handleClickOutside);
			document.removeEventListener('keydown', handleKeydown);
		};
	});
</script>

{#if $activeConnection}
	<div class="database-switcher" bind:this={dropdownRef} data-testid="database-switcher">
		<button
			class="switcher-trigger"
			class:switching={isSwitching}
			onclick={handleToggle}
			disabled={isSwitching}
			data-testid="database-switcher-trigger"
			title="Switch database"
		>
			<Icon name={isSwitching ? 'refresh' : 'database'} size={12} class={isSwitching ? 'spinning' : ''} />
			<span class="switcher-label">{$activeConnection.database}</span>
			<Icon name="chevron-down" size={12} class="switcher-chevron" />
		</button>

		{#if isOpen}
			<div class="switcher-menu" data-testid="database-switcher-menu">
				{#if isLoading}
					<div class="switcher-status">
						<Icon name="refresh" size={12} class="spinning" />
						Loading databases...
					</div>
				{:else if error}
					<div class="switcher-error">
						<Icon name="alert-circle" size={12} />
						{error}
					</div>
				{:else if databases.length === 0}
					<div class="switcher-status">No databases found</div>
				{:else}
					{#each databases as db}
						<button
							class="switcher-item"
							class:active={db.name === $activeConnection.database}
							onclick={() => handleSelect(db.name)}
							data-testid="database-option-{db.name}"
						>
							<Icon name="database" size={12} />
							<span class="switcher-item-name">{db.name}</span>
							{#if db.size}
								<span class="switcher-item-meta">{db.size}</span>
							{/if}
						</button>
					{/each}
				{/if}
			</div>
		{/if}
	</div>
{/if}

<style>
	.database-switcher {
		position: relative;
		padding: 6px 12px;
		border-bottom: 1px solid var(--color-border);
	}

	.switcher-trigger {
		display: flex;
		align-items: center;
		gap: 6px;
		width: 100%;
		padding: 6px 10px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		cursor: pointer;
		text-align: left;
		font-size: 12px;
		transition: border-color var(--transition-fast);
	}

	.switcher-trigger:hover:not(:disabled) {
		border-color: var(--color-primary);
	}

	.switcher-trigger:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.switcher-trigger :global(svg) {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.switcher-label {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 500;
	}

	.switcher-chevron {
		flex-shrink: 0;
	}

	.switcher-menu {
		position: absolute;
		top: 100%;
		left: 12px;
		right: 12px;
		margin-top: 4px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		z-index: 100;
		max-height: 320px;
		overflow-y: auto;
		padding: 4px;
	}

	.switcher-status,
	.switcher-error {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 10px 12px;
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.switcher-error {
		color: var(--color-error);
	}

	.switcher-item {
		display: flex;
		align-items: center;
		gap: 6px;
		width: 100%;
		padding: 6px 10px;
		font-size: 12px;
		text-align: left;
		border-radius: var(--radius-sm);
		transition: background var(--transition-fast);
	}

	.switcher-item:hover {
		background: var(--color-surface);
	}

	.switcher-item.active {
		background: rgba(137, 180, 250, 0.12);
		color: var(--color-primary);
	}

	.switcher-item :global(svg) {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.switcher-item.active :global(svg) {
		color: var(--color-primary);
	}

	.switcher-item-name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.switcher-item-meta {
		font-size: 11px;
		color: var(--color-text-dim);
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
