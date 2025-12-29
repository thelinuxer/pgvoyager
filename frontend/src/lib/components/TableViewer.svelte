<script lang="ts">
	import { onMount } from 'svelte';
	import { activeConnectionId } from '$lib/stores/connections';
	import { tabs } from '$lib/stores/tabs';
	import { dataApi } from '$lib/api/client';
	import type { Tab, TableDataResponse, ColumnInfo, ForeignKeyPreview } from '$lib/types';
	import FKPreviewPopup from './FKPreviewPopup.svelte';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let data = $state<TableDataResponse | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	let page = $state(1);
	let pageSize = $state(100);
	let orderBy = $state<string | null>(null);
	let orderDir = $state<'ASC' | 'DESC'>('ASC');

	// FK Preview state
	let fkPreview = $state<ForeignKeyPreview | null>(null);
	let fkPreviewPosition = $state({ x: 0, y: 0 });
	let fkPreviewLoading = $state(false);
	let hoverTimeout: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		if (tab.schema && tab.table) {
			loadData();
		}
	});

	async function loadData() {
		if (!$activeConnectionId || !tab.schema || !tab.table) return;

		isLoading = true;
		error = null;

		try {
			data = await dataApi.getTableData($activeConnectionId, tab.schema, tab.table, {
				page,
				pageSize,
				orderBy: orderBy || undefined,
				orderDir
			});
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load data';
		} finally {
			isLoading = false;
		}
	}

	function handleSort(column: string) {
		if (orderBy === column) {
			orderDir = orderDir === 'ASC' ? 'DESC' : 'ASC';
		} else {
			orderBy = column;
			orderDir = 'ASC';
		}
		page = 1;
		loadData();
	}

	function handlePageChange(newPage: number) {
		page = newPage;
		loadData();
	}

	function handleFKClick(col: ColumnInfo, value: unknown) {
		if (!col.fkReference || value === null) return;

		tabs.openTable(col.fkReference.schema, col.fkReference.table);
	}

	async function handleFKHover(e: MouseEvent, col: ColumnInfo, value: unknown) {
		if (!col.fkReference || value === null || !$activeConnectionId) return;

		// Clear any existing timeout
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
		}

		// Set position
		fkPreviewPosition = { x: e.clientX, y: e.clientY };

		// Delay before fetching to avoid excessive API calls
		hoverTimeout = setTimeout(async () => {
			fkPreviewLoading = true;
			try {
				fkPreview = await dataApi.getForeignKeyPreview(
					$activeConnectionId!,
					col.fkReference!.schema,
					col.fkReference!.table,
					col.fkReference!.column,
					String(value)
				);
			} catch (e) {
				console.error('Failed to load FK preview:', e);
				fkPreview = null;
			} finally {
				fkPreviewLoading = false;
			}
		}, 300);
	}

	function handleFKLeave() {
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
			hoverTimeout = null;
		}
		fkPreview = null;
		fkPreviewLoading = false;
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

<div class="table-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<span class="table-name">{tab.schema}.{tab.table}</span>
			{#if data}
				<span class="row-count">{data.totalRows.toLocaleString()} rows</span>
			{/if}
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={loadData} disabled={isLoading}>
				🔄 Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !data}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if data}
		<div class="table-container">
			<table class="data-table">
				<thead>
					<tr>
						{#each data.columns as col}
							<th
								class:sortable={true}
								class:sorted={orderBy === col.name}
								onclick={() => handleSort(col.name)}
							>
								<div class="th-content">
									{#if col.isPrimaryKey}
										<span class="pk-icon" title="Primary Key">🔑</span>
									{/if}
									{#if col.isForeignKey}
										<span class="fk-icon" title="Foreign Key">🔗</span>
									{/if}
									<span class="col-name">{col.name}</span>
									{#if orderBy === col.name}
										<span class="sort-icon">{orderDir === 'ASC' ? '▲' : '▼'}</span>
									{/if}
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
								<td
									class:pk-column={col.isPrimaryKey}
									class:fk-column={col.isForeignKey && row[col.name] !== null}
									class:null-value={row[col.name] === null}
									onclick={() => col.isForeignKey && handleFKClick(col, row[col.name])}
									onmouseenter={(e) => handleFKHover(e, col, row[col.name])}
									onmouseleave={handleFKLeave}
								>
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
					onclick={() => handlePageChange(1)}
				>
					⟪
				</button>
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
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === data.totalPages}
					onclick={() => handlePageChange(data!.totalPages)}
				>
					⟫
				</button>
			</div>
			<div class="page-size">
				<select
					value={pageSize}
					onchange={(e) => {
						pageSize = parseInt(e.currentTarget.value);
						page = 1;
						loadData();
					}}
				>
					<option value={50}>50 rows</option>
					<option value={100}>100 rows</option>
					<option value={250}>250 rows</option>
					<option value={500}>500 rows</option>
					<option value={1000}>1000 rows</option>
				</select>
			</div>
		</div>
	{/if}
</div>

{#if fkPreview || fkPreviewLoading}
	<FKPreviewPopup
		preview={fkPreview}
		loading={fkPreviewLoading}
		x={fkPreviewPosition.x}
		y={fkPreviewPosition.y}
	/>
{/if}

<style>
	.table-viewer {
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

	.table-name {
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

	.pk-icon,
	.fk-icon {
		font-size: 10px;
	}

	th.sortable {
		cursor: pointer;
	}

	th.sortable:hover {
		background: var(--color-surface);
	}

	.sort-icon {
		font-size: 10px;
		color: var(--color-primary);
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

	.page-size select {
		padding: 4px 8px;
		font-size: 12px;
	}
</style>
