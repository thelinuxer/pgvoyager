<script lang="ts">
	import { activeConnectionId } from '$lib/stores/connections';
	import { schemaApi } from '$lib/api/client';
	import type { Tab, Sequence } from '$lib/types';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let sequenceInfo = $state<Sequence | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (tab.schema && tab.sequenceName) {
			loadSequence();
		}
	});

	async function loadSequence() {
		if (!$activeConnectionId || !tab.schema || !tab.sequenceName) return;

		isLoading = true;
		error = null;

		try {
			const sequences = await schemaApi.listSequences($activeConnectionId, tab.schema);
			sequenceInfo = sequences.find((s) => s.name === tab.sequenceName) || null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load sequence';
		} finally {
			isLoading = false;
		}
	}

	function formatNumber(num: number): string {
		if (num === 9223372036854775807) return 'MAX BIGINT';
		if (num === -9223372036854775808) return 'MIN BIGINT';
		return num.toLocaleString();
	}
</script>

<div class="sequence-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<div class="breadcrumb">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 2v20M2 12h20"/>
					<path d="M12 2l4 4-4 4"/>
				</svg>
				<span class="sequence-name">{tab.schema}.{tab.sequenceName}</span>
			</div>
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={loadSequence} disabled={isLoading}>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class:spinning={isLoading}>
					<path d="M23 4v6h-6M1 20v-6h6"/>
					<path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
				</svg>
				Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !sequenceInfo}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if sequenceInfo}
		<div class="content">
			<div class="info-section">
				<h3>Sequence Details</h3>
				<div class="info-grid">
					<div class="info-row">
						<span class="info-label">Name</span>
						<span class="info-value">{sequenceInfo.name}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Schema</span>
						<span class="info-value">{sequenceInfo.schema}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Owner</span>
						<span class="info-value">{sequenceInfo.owner}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Data Type</span>
						<span class="info-value mono">{sequenceInfo.dataType}</span>
					</div>
				</div>
			</div>

			<div class="info-section">
				<h3>Configuration</h3>
				<div class="info-grid">
					<div class="info-row">
						<span class="info-label">Start Value</span>
						<span class="info-value mono">{formatNumber(sequenceInfo.startValue)}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Min Value</span>
						<span class="info-value mono">{formatNumber(sequenceInfo.minValue)}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Max Value</span>
						<span class="info-value mono">{formatNumber(sequenceInfo.maxValue)}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Increment</span>
						<span class="info-value mono">{formatNumber(sequenceInfo.increment)}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Cache Size</span>
						<span class="info-value mono">{formatNumber(sequenceInfo.cacheSize)}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Cycle</span>
						<span class="info-value">{sequenceInfo.isCycled ? 'Yes' : 'No'}</span>
					</div>
				</div>
			</div>

			<div class="info-section">
				<h3>Current State</h3>
				<div class="info-grid">
					<div class="info-row">
						<span class="info-label">Last Value</span>
						<span class="info-value mono">
							{sequenceInfo.lastValue !== undefined && sequenceInfo.lastValue !== null
								? formatNumber(sequenceInfo.lastValue)
								: 'Not yet used'}
						</span>
					</div>
				</div>
			</div>
		</div>
	{:else}
		<div class="not-found">Sequence not found</div>
	{/if}
</div>

<style>
	.sequence-viewer {
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

	.sequence-name {
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

	.info-value.mono {
		font-family: var(--font-mono);
	}
</style>
