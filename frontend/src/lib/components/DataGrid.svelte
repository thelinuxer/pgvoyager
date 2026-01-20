<script lang="ts">
	import type { ColumnInfo } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		columns: ColumnInfo[];
		rows: Record<string, unknown>[];
		totalRows: number;
		page: number;
		pageSize: number;
		totalPages: number;

		// Sorting
		orderBy?: string | null;
		orderDir?: 'ASC' | 'DESC';
		onSort?: (column: string) => void;

		// Pagination
		onPageChange?: (page: number) => void;
		onPageSizeChange?: (pageSize: number) => void;

		// Export
		showExport?: boolean;
		onExport?: () => void;

		// Edit mode
		editMode?: boolean;
		hasPrimaryKey?: boolean;
		selectedRows?: Set<number>;
		editingCell?: { rowIndex: number; colName: string } | null;
		editValue?: string;
		onToggleEditMode?: () => void;
		onCellDoubleClick?: (rowIndex: number, colName: string, value: unknown) => void;
		onCellKeydown?: (e: KeyboardEvent, rowIndex: number, colName: string) => void;
		onCellBlur?: (rowIndex: number, colName: string) => void;
		onEditValueChange?: (value: string) => void;
		onToggleRowSelection?: (rowIndex: number) => void;
		onToggleSelectAll?: () => void;
		onDeleteSelected?: () => void;
		onAddRow?: () => void;
		isSaving?: boolean;

		// FK handling
		onFKClick?: (col: ColumnInfo, value: unknown) => void;
		onFKHover?: (e: MouseEvent, col: ColumnInfo, value: unknown) => void;
		onFKLeave?: () => void;

		// Loading state
		isLoading?: boolean;
	}

	let {
		columns,
		rows,
		totalRows,
		page,
		pageSize,
		totalPages,
		orderBy = null,
		orderDir = 'ASC',
		onSort,
		onPageChange,
		onPageSizeChange,
		showExport = false,
		onExport,
		editMode = false,
		hasPrimaryKey = false,
		selectedRows = new Set(),
		editingCell = null,
		editValue = '',
		onToggleEditMode,
		onCellDoubleClick,
		onCellKeydown,
		onCellBlur,
		onEditValueChange,
		onToggleRowSelection,
		onToggleSelectAll,
		onDeleteSelected,
		onAddRow,
		isSaving = false,
		onFKClick,
		onFKHover,
		onFKLeave,
		isLoading = false
	}: Props = $props();

	// Derived values for pagination display
	let showingStart = $derived((page - 1) * pageSize + 1);
	let showingEnd = $derived(Math.min(page * pageSize, totalRows));

	function formatValue(value: unknown): string {
		if (value === null) return 'NULL';
		if (value === undefined) return '';
		if (typeof value === 'object') {
			return JSON.stringify(value);
		}
		return String(value);
	}

	function handleSort(column: string) {
		if (onSort) {
			onSort(column);
		}
	}

	function handleCellClick(col: ColumnInfo, value: unknown) {
		if (!editMode && col.isForeignKey && onFKClick) {
			onFKClick(col, value);
		}
	}

	function handleCellHover(e: MouseEvent, col: ColumnInfo, value: unknown) {
		if (!editMode && onFKHover) {
			onFKHover(e, col, value);
		}
	}

	function handleCellLeave() {
		if (!editMode && onFKLeave) {
			onFKLeave();
		}
	}

	// Action to focus input on mount
	function focusOnMount(node: HTMLInputElement) {
		node.focus();
		node.select();
	}
</script>

<div class="data-grid">
	<div class="table-container">
		<table class="data-table" class:edit-mode={editMode}>
			<thead>
				<tr>
					{#if editMode && onToggleRowSelection}
						<th class="checkbox-col">
							<input
								type="checkbox"
								checked={rows.length > 0 && selectedRows.size === rows.length}
								onchange={() => onToggleSelectAll?.()}
								title="Select all"
							/>
						</th>
					{/if}
					{#each columns as col}
						<th
							class:sortable={!!onSort}
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
				{#if isLoading}
					<tr>
						<td colspan={columns.length + (editMode && onToggleRowSelection ? 1 : 0)} class="loading-row">
							<Icon name="refresh" size={16} class="spinning" />
							Loading...
						</td>
					</tr>
				{:else if rows.length === 0}
					<tr>
						<td colspan={columns.length + (editMode && onToggleRowSelection ? 1 : 0)} class="empty-row">
							No data
						</td>
					</tr>
				{:else}
					{#each rows as row, rowIndex}
						<tr class:selected={editMode && selectedRows.has(rowIndex)}>
							{#if editMode && onToggleRowSelection}
								<td class="checkbox-col">
									<input
										type="checkbox"
										checked={selectedRows.has(rowIndex)}
										onchange={() => onToggleRowSelection?.(rowIndex)}
									/>
								</td>
							{/if}
							{#each columns as col}
								{@const isEditing = editingCell?.rowIndex === rowIndex && editingCell?.colName === col.name}
								<td
									class:pk-column={col.isPrimaryKey}
									class:fk-column={col.isForeignKey && row[col.name] !== null && !editMode}
									class:null-value={row[col.name] === null}
									class:editable={editMode && !col.isPrimaryKey}
									class:editing={isEditing}
									onclick={() => handleCellClick(col, row[col.name])}
									ondblclick={() => !col.isPrimaryKey && onCellDoubleClick?.(rowIndex, col.name, row[col.name])}
									onmouseenter={(e) => handleCellHover(e, col, row[col.name])}
									onmouseleave={handleCellLeave}
								>
									{#if isEditing}
										<!-- svelte-ignore a11y_autofocus -->
										<input
											type="text"
											class="cell-input"
											value={editValue}
											oninput={(e) => onEditValueChange?.(e.currentTarget.value)}
											onkeydown={(e) => onCellKeydown?.(e, rowIndex, col.name)}
											onblur={() => onCellBlur?.(rowIndex, col.name)}
											use:focusOnMount
										/>
									{:else}
										{formatValue(row[col.name])}
									{/if}
								</td>
							{/each}
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>

	<div class="pagination">
		<div class="pagination-info">
			{#if totalRows > 0}
				Showing {showingStart} - {showingEnd} of {totalRows.toLocaleString()}
			{:else}
				No rows
			{/if}
		</div>
		<div class="pagination-controls">
			<button
				class="btn btn-sm btn-ghost"
				disabled={page === 1}
				onclick={() => onPageChange?.(1)}
				title="First page"
			>
				<Icon name="chevrons-left" size={14} />
			</button>
			<button
				class="btn btn-sm btn-ghost"
				disabled={page === 1}
				onclick={() => onPageChange?.(page - 1)}
				title="Previous page"
			>
				<Icon name="chevron-left" size={14} />
			</button>
			<span class="page-info">Page {page} of {totalPages}</span>
			<button
				class="btn btn-sm btn-ghost"
				disabled={page === totalPages || totalPages === 0}
				onclick={() => onPageChange?.(page + 1)}
				title="Next page"
			>
				<Icon name="chevron-right" size={14} />
			</button>
			<button
				class="btn btn-sm btn-ghost"
				disabled={page === totalPages || totalPages === 0}
				onclick={() => onPageChange?.(totalPages)}
				title="Last page"
			>
				<Icon name="chevrons-right" size={14} />
			</button>
		</div>
		<div class="pagination-right">
			{#if showExport && onExport}
				<button class="btn btn-sm btn-ghost" onclick={onExport} title="Export to CSV">
					<Icon name="download" size={14} />
					Export CSV
				</button>
			{/if}
			<select
				value={pageSize}
				onchange={(e) => onPageSizeChange?.(parseInt(e.currentTarget.value))}
			>
				<option value={50}>50 rows</option>
				<option value={100}>100 rows</option>
				<option value={250}>250 rows</option>
				<option value={500}>500 rows</option>
				<option value={1000}>1000 rows</option>
			</select>
		</div>
	</div>
</div>

<style>
	.data-grid {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.table-container {
		flex: 1;
		overflow: auto;
	}

	/* Column header content */
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

	/* Pagination */
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

	.pagination-right {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.pagination-right select {
		padding: 4px 8px;
		font-size: 12px;
	}

	/* Edit mode styles */
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

	/* Loading and empty states */
	.loading-row,
	.empty-row {
		text-align: center;
		padding: 40px !important;
		color: var(--color-text-muted);
	}

	.loading-row {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
