<script lang="ts">
	import { activeConnectionId } from '$lib/stores/connections';
	import { schemaApi } from '$lib/api/client';
	import type { Tab, Function } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let functionInfo = $state<Function | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (tab.schema && tab.functionName) {
			loadFunction();
		}
	});

	async function loadFunction() {
		if (!$activeConnectionId || !tab.schema || !tab.functionName) return;

		isLoading = true;
		error = null;

		try {
			const functions = await schemaApi.listFunctions($activeConnectionId, tab.schema);
			functionInfo = functions.find((f) => f.name === tab.functionName) || null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load function';
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="function-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<div class="breadcrumb">
				<Icon name="terminal" size={14} />
				<span class="function-name">{tab.schema}.{tab.functionName}</span>
			</div>
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={loadFunction} disabled={isLoading}>
				<Icon name="refresh" size={14} class={isLoading ? 'spinning' : ''} />
				Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !functionInfo}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if functionInfo}
		<div class="content">
			<div class="info-section">
				<h3>Function Details</h3>
				<div class="info-grid">
					<div class="info-row">
						<span class="info-label">Name</span>
						<span class="info-value">{functionInfo.name}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Schema</span>
						<span class="info-value">{functionInfo.schema}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Owner</span>
						<span class="info-value">{functionInfo.owner}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Language</span>
						<span class="info-value">{functionInfo.language}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Return Type</span>
						<span class="info-value mono">{functionInfo.returnType}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Arguments</span>
						<span class="info-value mono">{functionInfo.arguments || '(none)'}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Is Aggregate</span>
						<span class="info-value">{functionInfo.isAggregate ? 'Yes' : 'No'}</span>
					</div>
					{#if functionInfo.comment}
						<div class="info-row">
							<span class="info-label">Comment</span>
							<span class="info-value">{functionInfo.comment}</span>
						</div>
					{/if}
				</div>
			</div>

			<div class="definition-section">
				<h3>Definition</h3>
				<pre class="definition">{functionInfo.definition}</pre>
			</div>
		</div>
	{:else}
		<div class="not-found">Function not found</div>
	{/if}
</div>

<style>
	.function-viewer {
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

	.function-name {
		font-weight: 600;
		font-family: var(--font-mono);
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

	.info-section h3,
	.definition-section h3 {
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

	.info-value.mono {
		font-family: var(--font-mono);
	}

	.definition {
		font-family: var(--font-mono);
		font-size: 13px;
		white-space: pre-wrap;
		background: var(--color-bg-tertiary);
		padding: 16px;
		border-radius: var(--radius-md);
		border: 1px solid var(--color-border);
		overflow: auto;
	}
</style>
