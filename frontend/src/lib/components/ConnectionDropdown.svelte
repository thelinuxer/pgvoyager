<script lang="ts">
	import { connections, activeConnectionId, activeConnection } from '$lib/stores/connections';
	import Icon from '$lib/icons/Icon.svelte';
	import type { Connection } from '$lib/types';

	interface Props {
		isConnecting: boolean;
		onConnect: (id: string) => void;
		onEdit: (connection: Connection) => void;
	}

	let { isConnecting, onConnect, onEdit }: Props = $props();

	let isOpen = $state(false);
	let dropdownRef = $state<HTMLDivElement | null>(null);

	function handleToggle() {
		if (!isConnecting) {
			isOpen = !isOpen;
		}
	}

	function handleSelect(id: string) {
		isOpen = false;
		onConnect(id);
	}

	function handleEdit(e: MouseEvent, connection: Connection) {
		e.stopPropagation();
		isOpen = false;
		onEdit(connection);
	}

	function handleClickOutside(e: MouseEvent) {
		if (dropdownRef && !dropdownRef.contains(e.target as Node)) {
			isOpen = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			isOpen = false;
		}
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

<div class="connection-dropdown" bind:this={dropdownRef}>
	<button
		class="dropdown-trigger"
		class:connecting={isConnecting}
		onclick={handleToggle}
		disabled={isConnecting}
	>
		{#if isConnecting}
			<Icon name="refresh" size={14} class="spinning" />
		{:else}
			<Icon name="database" size={14} />
		{/if}
		<span class="selected-text">
			{#if isConnecting}
				Connecting...
			{:else if $activeConnection}
				{$activeConnection.name} ({$activeConnection.host}:{$activeConnection.port}/{$activeConnection.database})
			{:else}
				Select Connection...
			{/if}
		</span>
		<Icon name="chevron-down" size={14} class="chevron" />
	</button>

	{#if isOpen}
		<div class="dropdown-menu">
			{#if $connections.length === 0}
				<div class="dropdown-empty">
					No connections configured
				</div>
			{:else}
				{#each $connections as conn}
					<div class="dropdown-item" class:active={$activeConnectionId === conn.id}>
						<button
							class="item-main"
							onclick={() => handleSelect(conn.id)}
						>
							<span class="item-name">{conn.name}</span>
							<span class="item-details">{conn.host}:{conn.port}/{conn.database}</span>
							{#if conn.isConnected}
								<span class="connected-indicator" title="Connected">●</span>
							{/if}
						</button>
						<button
							class="item-edit"
							onclick={(e) => handleEdit(e, conn)}
							title="Edit connection"
						>
							<Icon name="edit" size={12} />
						</button>
					</div>
				{/each}
			{/if}
		</div>
	{/if}
</div>

<style>
	.connection-dropdown {
		position: relative;
	}

	.dropdown-trigger {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 12px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		cursor: pointer;
		min-width: 320px;
		text-align: left;
	}

	.dropdown-trigger:hover:not(:disabled) {
		border-color: var(--color-border-hover, var(--color-border));
	}

	.dropdown-trigger:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.dropdown-trigger.connecting {
		border-color: var(--color-primary);
	}

	.dropdown-trigger :global(svg) {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.selected-text {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.chevron {
		transition: transform 0.2s ease;
	}

	.dropdown-menu {
		position: absolute;
		top: 100%;
		left: 0;
		right: 0;
		margin-top: 4px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		z-index: 1000;
		max-height: 300px;
		overflow-y: auto;
	}

	.dropdown-empty {
		padding: 12px 16px;
		color: var(--color-text-muted);
		text-align: center;
		font-size: 13px;
	}

	.dropdown-item {
		display: flex;
		align-items: center;
		border-bottom: 1px solid var(--color-border);
	}

	.dropdown-item:last-child {
		border-bottom: none;
	}

	.dropdown-item.active {
		background: rgba(137, 180, 250, 0.1);
	}

	.item-main {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		background: transparent;
		border: none;
		cursor: pointer;
		text-align: left;
		min-width: 0;
	}

	.item-main:hover {
		background: var(--color-bg-hover, rgba(255, 255, 255, 0.05));
	}

	.item-name {
		font-weight: 500;
		white-space: nowrap;
	}

	.item-details {
		color: var(--color-text-muted);
		font-size: 12px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.connected-indicator {
		color: var(--color-success);
		font-size: 10px;
		flex-shrink: 0;
	}

	.item-edit {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 8px 12px;
		background: transparent;
		border: none;
		border-left: 1px solid var(--color-border);
		cursor: pointer;
		color: var(--color-text-muted);
	}

	.item-edit:hover {
		background: var(--color-bg-hover, rgba(255, 255, 255, 0.05));
		color: var(--color-text);
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
