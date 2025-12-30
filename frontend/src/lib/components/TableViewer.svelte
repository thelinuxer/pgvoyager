<script lang="ts">
	import { onMount } from 'svelte';
	import { activeConnectionId } from '$lib/stores/connections';
	import { tabs } from '$lib/stores/tabs';
	import { dataApi } from '$lib/api/client';
	import type { Tab, TableDataResponse, ColumnInfo, ForeignKeyPreview, TableLocation } from '$lib/types';
	import FKPreviewPopup from './FKPreviewPopup.svelte';
	import Icon from '$lib/icons/Icon.svelte';

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

	// CRUD state
	let editMode = $state(false);
	let editingCell = $state<{ rowIndex: number; colName: string } | null>(null);
	let editValue = $state('');
	let selectedRows = $state<Set<number>>(new Set());
	let showAddRowModal = $state(false);
	let newRowData = $state<Record<string, string>>({});
	let isSaving = $state(false);
	let crudError = $state<string | null>(null);

	// Check if table has primary key
	let hasPrimaryKey = $derived(data?.columns.some((c) => c.isPrimaryKey) ?? false);
	let primaryKeyColumns = $derived(data?.columns.filter((c) => c.isPrimaryKey) ?? []);

	// FK Preview state
	let fkPreview = $state<ForeignKeyPreview | null>(null);
	let fkPreviewPosition = $state({ x: 0, y: 0 });
	let fkPreviewLoading = $state(false);
	let hoverTimeout: ReturnType<typeof setTimeout> | null = null;
	let closeTimeout: ReturnType<typeof setTimeout> | null = null;

	// Navigation state
	let canGoBack = $derived(tabs.canNavigateBack(tab.id));
	let canGoForward = $derived(tabs.canNavigateForward(tab.id));
	let currentLocation = $derived(tabs.getCurrentLocation(tab.id));

	// Reload data when tab's schema/table changes
	$effect(() => {
		if (tab.schema && tab.table) {
			// Reset pagination when navigating
			page = 1;
			// Reset edit mode state
			editMode = false;
			editingCell = null;
			selectedRows = new Set();
			crudError = null;
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

		// Clear any existing timeouts
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
		}
		if (closeTimeout) {
			clearTimeout(closeTimeout);
			closeTimeout = null;
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
		// Delay closing to allow mouse to move to popup
		closeTimeout = setTimeout(() => {
			fkPreview = null;
			fkPreviewLoading = false;
		}, 150);
	}

	function handlePopupEnter() {
		// Cancel close when mouse enters popup
		if (closeTimeout) {
			clearTimeout(closeTimeout);
			closeTimeout = null;
		}
	}

	function handlePopupLeave() {
		// Close popup when mouse leaves it
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

	// Action to focus input on mount
	function focusOnMount(node: HTMLInputElement) {
		node.focus();
		node.select();
	}

	// CRUD functions
	function toggleEditMode() {
		editMode = !editMode;
		if (!editMode) {
			// Clear edit state when exiting edit mode
			editingCell = null;
			selectedRows = new Set();
			crudError = null;
		}
	}

	function handleCellDoubleClick(rowIndex: number, colName: string, value: unknown) {
		if (!editMode || !hasPrimaryKey) return;
		editingCell = { rowIndex, colName };
		editValue = value === null ? '' : String(value);
	}

	function handleCellKeydown(e: KeyboardEvent, rowIndex: number, colName: string) {
		if (e.key === 'Enter') {
			e.preventDefault();
			saveCell(rowIndex, colName);
		} else if (e.key === 'Escape') {
			editingCell = null;
		}
	}

	async function saveCell(rowIndex: number, colName: string) {
		if (!$activeConnectionId || !tab.schema || !tab.table || !data) return;

		const row = data.rows[rowIndex];
		const pkData: Record<string, unknown> = {};
		for (const pkCol of primaryKeyColumns) {
			pkData[pkCol.name] = row[pkCol.name];
		}

		// Convert empty string to null
		const newValue = editValue.trim() === '' ? null : editValue;

		isSaving = true;
		crudError = null;

		try {
			await dataApi.updateRow($activeConnectionId, tab.schema, tab.table, {
				primaryKey: pkData,
				data: { [colName]: newValue }
			});

			// Update local data
			data.rows[rowIndex][colName] = newValue;
			editingCell = null;
		} catch (e) {
			crudError = e instanceof Error ? e.message : 'Update failed';
		} finally {
			isSaving = false;
		}
	}

	function toggleRowSelection(rowIndex: number) {
		const newSet = new Set(selectedRows);
		if (newSet.has(rowIndex)) {
			newSet.delete(rowIndex);
		} else {
			newSet.add(rowIndex);
		}
		selectedRows = newSet;
	}

	function toggleSelectAll() {
		if (!data) return;
		if (selectedRows.size === data.rows.length) {
			selectedRows = new Set();
		} else {
			selectedRows = new Set(data.rows.map((_, i) => i));
		}
	}

	async function deleteSelectedRows() {
		if (!$activeConnectionId || !tab.schema || !tab.table || !data || selectedRows.size === 0) return;

		if (!confirm(`Delete ${selectedRows.size} row(s)? This cannot be undone.`)) return;

		isSaving = true;
		crudError = null;

		try {
			const rowsToDelete = Array.from(selectedRows).sort((a, b) => b - a); // Delete from end first

			for (const rowIndex of rowsToDelete) {
				const row = data.rows[rowIndex];
				const pkData: Record<string, unknown> = {};
				for (const pkCol of primaryKeyColumns) {
					pkData[pkCol.name] = row[pkCol.name];
				}

				await dataApi.deleteRow($activeConnectionId!, tab.schema!, tab.table!, {
					primaryKey: pkData
				});
			}

			selectedRows = new Set();
			await loadData();
		} catch (e) {
			crudError = e instanceof Error ? e.message : 'Delete failed';
		} finally {
			isSaving = false;
		}
	}

	function openAddRowModal() {
		if (!data) return;
		newRowData = {};
		for (const col of data.columns) {
			newRowData[col.name] = '';
		}
		showAddRowModal = true;
	}

	function closeAddRowModal() {
		showAddRowModal = false;
		newRowData = {};
	}

	async function addNewRow() {
		if (!$activeConnectionId || !tab.schema || !tab.table) return;

		// Filter out empty values
		const insertData: Record<string, unknown> = {};
		for (const [key, value] of Object.entries(newRowData)) {
			if (value.trim() !== '') {
				insertData[key] = value;
			}
		}

		if (Object.keys(insertData).length === 0) {
			crudError = 'Please enter at least one value';
			return;
		}

		isSaving = true;
		crudError = null;

		try {
			await dataApi.insertRow($activeConnectionId, tab.schema, tab.table, {
				data: insertData
			});

			closeAddRowModal();
			await loadData();
		} catch (e) {
			crudError = e instanceof Error ? e.message : 'Insert failed';
		} finally {
			isSaving = false;
		}
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
					<Icon name="arrow-left" size={16} />
				</button>
				<button
					class="btn btn-sm btn-ghost nav-btn"
					onclick={handleForward}
					disabled={!canGoForward}
					title="Go Forward (Alt+→)"
				>
					<Icon name="arrow-right" size={16} />
				</button>
			</div>
			<div class="breadcrumb">
				<Icon name="table" size={14} />
				<span class="table-name">{tab.schema}.{tab.table}</span>
			</div>
			{#if currentLocation?.filter}
				<div class="filter-badge" title="Filtered by {currentLocation.filter.column} = {currentLocation.filter.value}">
					<Icon name="filter" size={12} />
					<span class="filter-text">{currentLocation.filter.column} = {currentLocation.filter.value}</span>
					<button class="filter-clear" onclick={clearFilter} title="Clear filter">
						<Icon name="x" size={10} strokeWidth={3} />
					</button>
				</div>
			{/if}
			{#if data}
				<span class="row-count">{data.totalRows.toLocaleString()} rows</span>
			{/if}
		</div>
		<div class="toolbar-right">
			{#if editMode && selectedRows.size > 0}
				<button
					class="btn btn-sm btn-danger"
					onclick={deleteSelectedRows}
					disabled={isSaving}
				>
					<Icon name="trash" size={14} />
					Delete ({selectedRows.size})
				</button>
			{/if}
			{#if editMode}
				<button
					class="btn btn-sm btn-secondary"
					onclick={openAddRowModal}
					disabled={isSaving}
				>
					<Icon name="plus" size={14} />
					Add Row
				</button>
			{/if}
			<button
				class="btn btn-sm"
				class:btn-primary={editMode}
				class:btn-ghost={!editMode}
				onclick={toggleEditMode}
				disabled={!hasPrimaryKey && !editMode}
				title={!hasPrimaryKey ? 'Table has no primary key - editing disabled' : editMode ? 'Exit Edit Mode' : 'Enter Edit Mode'}
			>
				<Icon name="edit" size={14} />
				{editMode ? 'Done' : 'Edit'}
			</button>
			<button class="btn btn-sm btn-ghost" onclick={loadData} disabled={isLoading} title="Refresh">
				<Icon name="refresh" size={14} class={isLoading ? 'spinning' : ''} />
				Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !data}
		<div class="loading">
			<Icon name="refresh" size={24} class="spinning" />
			Loading...
		</div>
	{:else if error}
		<div class="error">
			<Icon name="alert-circle" size={20} />
			{error}
		</div>
	{:else if data}
		{#if crudError}
			<div class="crud-error">
				<Icon name="alert-circle" size={16} />
				{crudError}
				<button class="crud-error-close" onclick={() => crudError = null}>
					<Icon name="x" size={14} />
				</button>
			</div>
		{/if}
		<div class="table-container">
			<table class="data-table" class:edit-mode={editMode}>
				<thead>
					<tr>
						{#if editMode}
							<th class="checkbox-col">
								<input
									type="checkbox"
									checked={data.rows.length > 0 && selectedRows.size === data.rows.length}
									onchange={toggleSelectAll}
									title="Select all"
								/>
							</th>
						{/if}
						{#each data.columns as col}
							<th
								class:sortable={true}
								class:sorted={orderBy === col.name}
								onclick={() => handleSort(col.name)}
							>
								<div class="th-content">
									{#if col.isPrimaryKey}
										<span class="pk-icon" title="Primary Key">
											<Icon name="key" size={12} />
										</span>
									{/if}
									{#if col.isForeignKey}
										<span class="fk-icon" title="Foreign Key">
											<Icon name="link" size={12} />
										</span>
									{/if}
									<span class="col-name">{col.name}</span>
									{#if orderBy === col.name}
										<Icon name={orderDir === 'ASC' ? 'arrow-up' : 'arrow-down'} size={12} class="sort-icon" />
									{/if}
								</div>
								<div class="col-type">{col.dataType}</div>
							</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each data.rows as row, rowIndex}
						<tr class:selected={editMode && selectedRows.has(rowIndex)}>
							{#if editMode}
								<td class="checkbox-col">
									<input
										type="checkbox"
										checked={selectedRows.has(rowIndex)}
										onchange={() => toggleRowSelection(rowIndex)}
									/>
								</td>
							{/if}
							{#each data.columns as col}
								{@const isEditing = editingCell?.rowIndex === rowIndex && editingCell?.colName === col.name}
								<td
									class:pk-column={col.isPrimaryKey}
									class:fk-column={col.isForeignKey && row[col.name] !== null}
									class:null-value={row[col.name] === null}
									class:editable={editMode && !col.isPrimaryKey}
									class:editing={isEditing}
									onclick={() => !editMode && col.isForeignKey && handleFKClick(col, row[col.name])}
									ondblclick={() => !col.isPrimaryKey && handleCellDoubleClick(rowIndex, col.name, row[col.name])}
									onmouseenter={(e) => !editMode && handleFKHover(e, col, row[col.name])}
									onmouseleave={() => !editMode && handleFKLeave()}
								>
									{#if isEditing}
										<!-- svelte-ignore a11y_autofocus -->
										<input
											type="text"
											class="cell-input"
											bind:value={editValue}
											onkeydown={(e) => handleCellKeydown(e, rowIndex, col.name)}
											onblur={() => saveCell(rowIndex, col.name)}
											use:focusOnMount
										/>
									{:else}
										{formatValue(row[col.name])}
									{/if}
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
					<Icon name="chevrons-left" size={14} />
				</button>
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === 1}
					onclick={() => handlePageChange(page - 1)}
					title="Previous page"
				>
					<Icon name="chevron-left" size={14} />
				</button>
				<span class="page-info">Page {page} of {data.totalPages}</span>
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === data.totalPages}
					onclick={() => handlePageChange(page + 1)}
					title="Next page"
				>
					<Icon name="chevron-right" size={14} />
				</button>
				<button
					class="btn btn-sm btn-ghost"
					disabled={page === data.totalPages}
					onclick={() => handlePageChange(data!.totalPages)}
					title="Last page"
				>
					<Icon name="chevrons-right" size={14} />
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
		onMouseEnter={handlePopupEnter}
		onMouseLeave={handlePopupLeave}
	/>
{/if}

{#if showAddRowModal && data}
	<div class="modal-overlay" onclick={closeAddRowModal}>
		<div class="add-row-modal" onclick={(e) => e.stopPropagation()}>
			<div class="modal-header">
				<h3>Add New Row</h3>
				<button class="modal-close" onclick={closeAddRowModal}>
					<Icon name="x" size={18} />
				</button>
			</div>
			<div class="modal-body">
				{#if crudError}
					<div class="modal-error">
						<Icon name="alert-circle" size={14} />
						{crudError}
					</div>
				{/if}
				<div class="add-row-fields">
					{#each data.columns as col}
						<div class="field-row">
							<label for="field-{col.name}">
								<span class="field-name">{col.name}</span>
								<span class="field-type">{col.dataType}</span>
								{#if col.isPrimaryKey}
									<span class="field-badge pk">PK</span>
								{/if}
							</label>
							<input
								id="field-{col.name}"
								type="text"
								placeholder={col.isPrimaryKey ? 'Auto-generated if empty' : 'Enter value...'}
								bind:value={newRowData[col.name]}
							/>
						</div>
					{/each}
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn btn-ghost" onclick={closeAddRowModal} disabled={isSaving}>
					Cancel
				</button>
				<button class="btn btn-primary" onclick={addNewRow} disabled={isSaving}>
					{#if isSaving}
						<Icon name="refresh" size={14} class="spinning" />
						Inserting...
					{:else}
						Insert Row
					{/if}
				</button>
			</div>
		</div>
	</div>
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
		display: flex;
		color: var(--color-warning);
	}

	.fk-icon {
		display: flex;
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

	/* CRUD Styles */
	.crud-error {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: rgba(243, 139, 168, 0.15);
		border-bottom: 1px solid var(--color-error);
		color: var(--color-error);
		font-size: 13px;
	}

	.crud-error-close {
		margin-left: auto;
		padding: 2px;
		border-radius: 4px;
		opacity: 0.7;
		transition: opacity var(--transition-fast);
	}

	.crud-error-close:hover {
		opacity: 1;
	}

	.checkbox-col {
		width: 40px;
		text-align: center;
		padding: 8px !important;
	}

	.checkbox-col input[type="checkbox"] {
		width: 16px;
		height: 16px;
		cursor: pointer;
	}

	tr.selected {
		background: rgba(137, 180, 250, 0.15) !important;
	}

	tr.selected:hover {
		background: rgba(137, 180, 250, 0.2) !important;
	}

	.data-table.edit-mode td.editable {
		cursor: text;
	}

	.data-table.edit-mode td.editable:hover {
		background: rgba(137, 180, 250, 0.1);
	}

	td.editing {
		padding: 0 !important;
	}

	.cell-input {
		width: 100%;
		height: 100%;
		padding: 8px 12px;
		border: 2px solid var(--color-primary);
		background: var(--color-bg);
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: 13px;
		outline: none;
	}

	/* Add Row Modal */
	.modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.add-row-modal {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		width: 500px;
		max-width: 90vw;
		max-height: 80vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 16px 48px rgba(0, 0, 0, 0.3);
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid var(--color-border);
	}

	.modal-header h3 {
		font-size: 16px;
		font-weight: 600;
		margin: 0;
	}

	.modal-close {
		padding: 4px;
		border-radius: 4px;
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}

	.modal-close:hover {
		background: var(--color-surface);
		color: var(--color-text);
	}

	.modal-body {
		flex: 1;
		overflow-y: auto;
		padding: 20px;
	}

	.modal-error {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		background: rgba(243, 139, 168, 0.15);
		border: 1px solid var(--color-error);
		border-radius: var(--radius-md);
		color: var(--color-error);
		font-size: 13px;
		margin-bottom: 16px;
	}

	.add-row-fields {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.field-row {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.field-row label {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 13px;
	}

	.field-name {
		font-weight: 500;
		font-family: var(--font-mono);
	}

	.field-type {
		color: var(--color-text-dim);
		font-size: 11px;
	}

	.field-badge {
		font-size: 9px;
		font-weight: 600;
		padding: 2px 4px;
		border-radius: 3px;
		text-transform: uppercase;
	}

	.field-badge.pk {
		background: rgba(249, 226, 175, 0.2);
		color: var(--color-warning);
	}

	.field-row input {
		padding: 8px 12px;
		font-size: 13px;
		font-family: var(--font-mono);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-bg-secondary);
		color: var(--color-text);
		transition: border-color var(--transition-fast);
	}

	.field-row input:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.field-row input::placeholder {
		color: var(--color-text-dim);
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 16px 20px;
		border-top: 1px solid var(--color-border);
	}
</style>
