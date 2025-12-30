<script lang="ts">
	import { activeConnectionId } from '$lib/stores/connections';
	import { schemaApi } from '$lib/api/client';
	import type { Tab, CustomType } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let typeInfo = $state<CustomType | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (tab.schema && tab.typeName) {
			loadType();
		}
	});

	async function loadType() {
		if (!$activeConnectionId || !tab.schema || !tab.typeName) return;

		isLoading = true;
		error = null;

		try {
			const types = await schemaApi.listTypes($activeConnectionId, tab.schema);
			typeInfo = types.find((t) => t.name === tab.typeName) || null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load type';
		} finally {
			isLoading = false;
		}
	}

	function getTypeLabel(type: string): string {
		switch (type) {
			case 'enum': return 'Enumeration';
			case 'composite': return 'Composite';
			case 'domain': return 'Domain';
			case 'range': return 'Range';
			default: return type;
		}
	}
</script>

<div class="type-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<div class="breadcrumb">
				<Icon name="type" size={14} />
				<span class="type-name">{tab.schema}.{tab.typeName}</span>
			</div>
			{#if typeInfo}
				<span class="type-badge">{getTypeLabel(typeInfo.type)}</span>
			{/if}
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={loadType} disabled={isLoading}>
				<Icon name="refresh" size={14} class={isLoading ? 'spinning' : ''} />
				Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !typeInfo}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if typeInfo}
		<div class="content">
			<div class="info-section">
				<h3>Type Details</h3>
				<div class="info-grid">
					<div class="info-row">
						<span class="info-label">Name</span>
						<span class="info-value">{typeInfo.name}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Schema</span>
						<span class="info-value">{typeInfo.schema}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Owner</span>
						<span class="info-value">{typeInfo.owner}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Type</span>
						<span class="info-value">{getTypeLabel(typeInfo.type)}</span>
					</div>
					{#if typeInfo.comment}
						<div class="info-row">
							<span class="info-label">Comment</span>
							<span class="info-value">{typeInfo.comment}</span>
						</div>
					{/if}
				</div>
			</div>

			{#if typeInfo.type === 'enum' && typeInfo.elements && typeInfo.elements.length > 0}
				<div class="info-section">
					<h3>Enum Values</h3>
					<div class="elements-list">
						{#each typeInfo.elements as element, index}
							<div class="element-item">
								<span class="element-index">{index + 1}</span>
								<span class="element-value">{element}</span>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<div class="not-found">Type not found</div>
	{/if}
</div>

<style>
	.type-viewer {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 16px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
	}

	.toolbar-left {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.breadcrumb svg {
		color: var(--color-text-muted);
	}

	.type-name {
		font-weight: 600;
		font-family: var(--font-mono);
	}

	.type-badge {
		font-size: 11px;
		padding: 2px 8px;
		background: var(--color-primary);
		color: var(--color-bg);
		border-radius: var(--radius-sm);
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.loading,
	.error,
	.not-found {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--color-text-muted);
	}

	.error {
		color: var(--color-error);
	}

	.content {
		flex: 1;
		overflow: auto;
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.info-section h3 {
		font-size: 14px;
		font-weight: 600;
		color: var(--color-text-muted);
		margin-bottom: 12px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.info-grid {
		display: flex;
		flex-direction: column;
		gap: 8px;
		background: var(--color-bg-secondary);
		padding: 16px;
		border-radius: var(--radius-md);
		border: 1px solid var(--color-border);
	}

	.info-row {
		display: flex;
		gap: 16px;
	}

	.info-label {
		min-width: 120px;
		color: var(--color-text-muted);
		font-size: 13px;
	}

	.info-value {
		flex: 1;
		font-size: 13px;
	}

	.elements-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
		background: var(--color-bg-secondary);
		padding: 16px;
		border-radius: var(--radius-md);
		border: 1px solid var(--color-border);
	}

	.element-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 12px;
		background: var(--color-bg-tertiary);
		border-radius: var(--radius-sm);
	}

	.element-index {
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--color-surface);
		border-radius: 50%;
		font-size: 11px;
		color: var(--color-text-muted);
	}

	.element-value {
		font-family: var(--font-mono);
		font-size: 13px;
	}
</style>
