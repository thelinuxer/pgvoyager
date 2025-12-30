<script lang="ts">
	import { onMount } from 'svelte';
	import { activeConnectionId } from '$lib/stores/connections';
	import { tabs } from '$lib/stores/tabs';
	import { dataApi } from '$lib/api/client';
	import type { Tab, TableDataResponse, ColumnInfo, ForeignKeyPreview, TableLocation } from '$lib/types';
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

	// Navigation state
	let canGoBack = $derived(tabs.canNavigateBack(tab.id));
	let canGoForward = $derived(tabs.canNavigateForward(tab.id));
	let currentLocation = $derived(tabs.getCurrentLocation(tab.id));

	// Reload data when tab's schema/table changes
	$effect(() => {
		if (tab.schema && tab.table) {
			// Reset pagination when navigating
			page = 1;
			// Use sort from location if available
			const location = tabs.getCurrentLocation(tab.id);
			if (location?.sort) {
				orderBy = location.sort.column;
				orderDir = location.sort.direction;
			} else {
				orderBy = null;
				orderDir = 'ASC';
			}
			// Use limit from location if specified
			if (location?.limit) {
				pageSize = location.limit;
			}
			loadData();
		}
	});

	async function loadData() {
		if (!$activeConnectionId || !tab.schema || !tab.table) return;

		isLoading = true;
		error = null;

		try {
			const location = currentLocation;

			// Apply filter and sort from navigation if present
			data = await dataApi.getTableData($activeConnectionId, tab.schema, tab.table, {
				page,
				pageSize,
				orderBy: orderBy || location?.sort?.column || undefined,
				orderDir: orderDir || location?.sort?.direction || 'ASC',
				filterColumn: location?.filter?.column,
				filterValue: location?.filter?.value
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

		// Handle FK click - respects tab pinning (pinned = new tab, unpinned = navigate within)
		tabs.handleFKClick(
			tab.id,
			col.fkReference.schema,
			col.fkReference.table,
			col.fkReference.column,
			String(value)
		);
	}

	function handleBack() {
		tabs.navigateBack(tab.id);
	}

	function handleForward() {
		tabs.navigateForward(tab.id);
	}

	function clearFilter() {
		// Navigate to the same table without filter
		tabs.navigateToFK(tab.id, tab.schema!, tab.table!);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.altKey && e.key === 'ArrowLeft' && canGoBack) {
			e.preventDefault();
			handleBack();
		}
		if (e.altKey && e.key === 'ArrowRight' && canGoForward) {
			e.preventDefault();
			handleForward();
		}
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

<svelte:window onkeydown={handleKeydown} />

<div class="table-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<div class="nav-buttons">
				<button
					class="btn btn-sm btn-ghost nav-btn"
					onclick={handleBack}
					disabled={!canGoBack}
					title="Go Back (Alt+←)"
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M19 12H5M12 19l-7-7 7-7"/>
					</svg>
				</button>
				<button
					class="btn btn-sm btn-ghost nav-btn"
					onclick={handleForward}
					disabled={!canGoForward}
					title="Go Forward (Alt+→)"
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M5 12h14M12 5l7 7-7 7"/>
					</svg>
				</button>
			</div>
			<div class="breadcrumb">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<rect x="3" y="3" width="18" height="18" rx="2"/>
					<path d="M3 9h18M9 21V9"/>
				</svg>
				<span class="table-name">{tab.schema}.{tab.table}</span>
			</div>
			{#if currentLocation?.filter}
				<div class="filter-badge" title="Filtered by {currentLocation.filter.column} = {currentLocation.filter.value}">
					<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/>
					</svg>
					<span class="filter-text">{currentLocation.filter.column} = {currentLocation.filter.value}</span>
					<button class="filter-clear" onclick={clearFilter} title="Clear filter">
						<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
							<path d="M18 6L6 18M6 6l12 12"/>
						</svg>
					</button>
				</div>
			{/if}
			{#if data}
				<span class="row-count">{data.totalRows.toLocaleString()} rows</span>
			{/if}
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={loadData} disabled={isLoading} title="Refresh">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class:spinning={isLoading}>
					<path d="M23 4v6h-6M1 20v-6h6"/>
					<path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
				</svg>
				Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !data}
		<div class="loading">
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spinning">
				<path d="M23 4v6h-6M1 20v-6h6"/>
				<path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
			</svg>
			Loading...
		</div>
	{:else if error}
		<div class="error">
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="12" cy="12" r="10"/>
				<path d="M12 8v4M12 16h.01"/>
			</svg>
			{error}
		</div>
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
										<svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" class="pk-icon" title="Primary Key">
											<path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
										</svg>
									{/if}
									{#if col.isForeignKey}
										<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="fk-icon" title="Foreign Key">
											<path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/>
											<path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/>
										</svg>
									{/if}
									<span class="col-name">{col.name}</span>
									{#if orderBy === col.name}
										<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="sort-icon">
											{#if orderDir === 'ASC'}
												<path d="M12 19V5M5 12l7-7 7 7"/>
											{:else}
												<path d="M12 5v14M5 12l7 7 7-7"/>
											{/if}
										</svg>
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
					title="First page"
				>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M11 17l-5-5 5-5M18 17l-5-5 5-5"/>
					</svg>
				</button>
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === 1}
					onclick={() => handlePageChange(page - 1)}
					title="Previous page"
				>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M15 18l-6-6 6-6"/>
					</svg>
				</button>
				<span class="page-info">Page {page} of {data.totalPages}</span>
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === data.totalPages}
					onclick={() => handlePageChange(page + 1)}
					title="Next page"
				>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M9 18l6-6-6-6"/>
					</svg>
				</button>
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === data.totalPages}
					onclick={() => handlePageChange(data!.totalPages)}
					title="Last page"
				>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M13 17l5-5-5-5M6 17l5-5-5-5"/>
					</svg>
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
		gap: 12px;
	}

	.toolbar-left {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.toolbar-right {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.nav-buttons {
		display: flex;
		gap: 2px;
	}

	.nav-btn {
		padding: 6px;
		min-width: 28px;
	}

	.nav-btn:disabled {
		opacity: 0.3;
	}

	.nav-btn svg {
		display: block;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.breadcrumb svg {
		color: var(--color-text-muted);
	}

	.table-name {
		font-weight: 600;
		font-family: var(--font-mono);
	}

	.row-count {
		font-size: 12px;
		color: var(--color-text-muted);
		padding: 2px 8px;
		background: var(--color-surface);
		border-radius: 10px;
	}

	.filter-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 4px 8px;
		background: rgba(137, 180, 250, 0.15);
		border: 1px solid var(--color-primary);
		border-radius: var(--radius-sm);
		font-size: 12px;
		color: var(--color-primary);
	}

	.filter-badge svg {
		flex-shrink: 0;
	}

	.filter-text {
		font-family: var(--font-mono);
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.filter-clear {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2px;
		border-radius: 2px;
		opacity: 0.7;
		transition: all var(--transition-fast);
	}

	.filter-clear:hover {
		opacity: 1;
		background: var(--color-primary);
		color: var(--color-bg);
	}

	.loading,
	.error {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		color: var(--color-text-muted);
	}

	.error {
		color: var(--color-error);
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
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

	.pk-icon {
		color: var(--color-warning);
	}

	.fk-icon {
		color: var(--color-primary);
	}

	th.sortable {
		cursor: pointer;
	}

	th.sortable:hover {
		background: var(--color-surface);
	}

	.sort-icon {
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

	.pagination-controls .btn {
		padding: 4px 6px;
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
