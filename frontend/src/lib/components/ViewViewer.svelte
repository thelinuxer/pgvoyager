<script lang="ts">
	import { onMount } from 'svelte';
	import { activeConnectionId } from '$lib/stores/connections';
	import { schemaApi, dataApi } from '$lib/api/client';
	import type { Tab, View, TableDataResponse } from '$lib/types';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let viewInfo = $state<View | null>(null);
	let data = $state<TableDataResponse | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let activeViewTab = $state<'data' | 'definition'>('data');

	let page = $state(1);
	let pageSize = $state(100);

	$effect(() => {
		if (tab.schema && tab.view) {
			loadView();
		}
	});

	async function loadView() {
		if (!$activeConnectionId || !tab.schema || !tab.view) return;

		isLoading = true;
		error = null;

		try {
			// Load view definition
			const views = await schemaApi.listViews($activeConnectionId, tab.schema);
			viewInfo = views.find((v) => v.name === tab.view) || null;

			// Load view data (views can be queried like tables)
			data = await dataApi.getTableData($activeConnectionId, tab.schema, tab.view!, {
				page,
				pageSize
			});
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load view';
		} finally {
			isLoading = false;
		}
	}

	function handlePageChange(newPage: number) {
		page = newPage;
		loadView();
	}

	function formatValue(value: unknown): string {
		if (value === null) return 'NULL';
		if (value === undefined) return '';
		if (typeof value === 'object') {
			return JSON.stringify(value);
		}
		return String(value);
	}
</script>

<div class="view-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<span class="view-name">{tab.schema}.{tab.view}</span>
			{#if data}
				<span class="row-count">{data.totalRows.toLocaleString()} rows</span>
			{/if}
		</div>
		<div class="toolbar-tabs">
			<button
				class="tab-btn"
				class:active={activeViewTab === 'data'}
				onclick={() => (activeViewTab = 'data')}
			>
				Data
			</button>
			<button
				class="tab-btn"
				class:active={activeViewTab === 'definition'}
				onclick={() => (activeViewTab = 'definition')}
			>
				Definition
			</button>
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={loadView} disabled={isLoading}>
				🔄 Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !data}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if activeViewTab === 'definition' && viewInfo}
		<div class="definition-container">
			<pre class="definition">{viewInfo.definition}</pre>
		</div>
	{:else if data}
		<div class="table-container">
			<table class="data-table">
				<thead>
					<tr>
						{#each data.columns as col}
							<th>
								<div class="th-content">
									<span class="col-name">{col.name}</span>
								</div>
								<div class="col-type">{col.dataType}</div>
							</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each data.rows as row}
						<tr>
							{#each data.columns as col}
								<td class:null-value={row[col.name] === null}>
									{formatValue(row[col.name])}
								</td>
							{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<div class="pagination">
			<div class="pagination-info">
				Showing {(page - 1) * pageSize + 1} - {Math.min(page * pageSize, data.totalRows)} of {data.totalRows.toLocaleString()}
			</div>
			<div class="pagination-controls">
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === 1}
					onclick={() => handlePageChange(page - 1)}
				>
					⟨
				</button>
				<span class="page-info">Page {page} of {data.totalPages}</span>
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === data.totalPages}
					onclick={() => handlePageChange(page + 1)}
				>
					⟩
				</button>
			</div>
		</div>
	{/if}
</div>

<style>
	.view-viewer {
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

	.toolbar-tabs {
		display: flex;
		gap: 4px;
	}

	.tab-btn {
		padding: 4px 12px;
		border-radius: var(--radius-sm);
		font-size: 12px;
		transition: all var(--transition-fast);
	}

	.tab-btn:hover {
		background: var(--color-surface);
	}

	.tab-btn.active {
		background: var(--color-primary);
		color: var(--color-bg);
	}

	.view-name {
		font-weight: 600;
		font-family: var(--font-mono);
	}

	.row-count {
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.loading,
	.error {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--color-text-muted);
	}

	.error {
		color: var(--color-error);
	}

	.table-container {
		flex: 1;
		overflow: auto;
	}

	.definition-container {
		flex: 1;
		overflow: auto;
		padding: 16px;
	}

	.definition {
		font-family: var(--font-mono);
		font-size: 13px;
		white-space: pre-wrap;
		background: var(--color-bg-tertiary);
		padding: 16px;
		border-radius: var(--radius-md);
	}

	.th-content {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.col-type {
		font-size: 10px;
		font-weight: normal;
		color: var(--color-text-dim);
		margin-top: 2px;
	}

	.pagination {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 16px;
		background: var(--color-bg-secondary);
		border-top: 1px solid var(--color-border);
	}

	.pagination-info {
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.pagination-controls {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.page-info {
		padding: 0 12px;
		font-size: 13px;
	}
</style>
