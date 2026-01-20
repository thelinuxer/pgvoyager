<script lang="ts">
	import { onMount, onDestroy, untrack } from 'svelte';
	import { activeConnectionId, activeConnection } from '$lib/stores/connections';
	import { queryApi } from '$lib/api/client';
	import { layout } from '$lib/stores/layout';
	import { queryHistory } from '$lib/stores/queryHistory';
	import { tables, views, functions, allColumns } from '$lib/stores/schema';
	import { editorStore } from '$lib/stores/editor';
	import { tabs } from '$lib/stores/tabs';
	import type { Tab, QueryResult } from '$lib/types';
	import CodeMirror from 'svelte-codemirror-editor';
	import { sql, PostgreSQL } from '@codemirror/lang-sql';
	import { autocompletion } from '@codemirror/autocomplete';
	import { createSchemaCompletionSource } from '$lib/utils/sqlAutocomplete';
	import { EditorView } from '@codemirror/view';
	import { StateEffect, StateField, RangeSetBuilder } from '@codemirror/state';
	import { Decoration, type DecorationSet } from '@codemirror/view';
	import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
	import { tags } from '@lezer/highlight';

	// Custom theme that follows app CSS variables
	const appTheme = EditorView.theme({
		'&': {
			backgroundColor: 'var(--color-bg)',
			color: 'var(--color-text)'
		},
		'.cm-content': {
			caretColor: 'var(--color-primary)',
			fontFamily: 'var(--font-mono)'
		},
		'.cm-cursor, .cm-dropCursor': {
			borderLeftColor: 'var(--color-primary)'
		},
		'&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
			backgroundColor: 'rgba(137, 180, 250, 0.5) !important'
		},
		'.cm-content ::selection': {
			backgroundColor: 'rgba(137, 180, 250, 0.5) !important'
		},
		'.cm-activeLine': {
			backgroundColor: 'transparent'
		},
		'.cm-gutters': {
			backgroundColor: 'var(--color-bg-secondary)',
			color: 'var(--color-text-dim)',
			border: 'none',
			borderRight: '1px solid var(--color-border)'
		},
		'.cm-activeLineGutter': {
			backgroundColor: 'var(--color-surface)'
		},
		'.cm-lineNumbers .cm-gutterElement': {
			padding: '0 8px'
		},
		'.cm-scroller': {
			overflow: 'auto'
		},
		'.cm-tooltip': {
			backgroundColor: 'var(--color-surface)',
			border: '1px solid var(--color-border)',
			color: 'var(--color-text)'
		},
		'.cm-tooltip-autocomplete': {
			'& > ul > li': {
				padding: '4px 8px'
			},
			'& > ul > li[aria-selected]': {
				backgroundColor: 'var(--color-primary)',
				color: 'var(--color-bg)'
			}
		}
	});

	// Syntax highlighting that matches app theme
	const appHighlightStyle = HighlightStyle.define([
		{ tag: tags.keyword, color: 'var(--color-primary)' },
		{ tag: tags.operator, color: 'var(--color-text)' },
		{ tag: tags.special(tags.variableName), color: 'var(--color-info)' },
		{ tag: tags.typeName, color: 'var(--color-warning)' },
		{ tag: tags.atom, color: 'var(--color-success)' },
		{ tag: tags.number, color: 'var(--color-success)' },
		{ tag: tags.string, color: 'var(--color-success)' },
		{ tag: tags.comment, color: 'var(--color-text-dim)', fontStyle: 'italic' },
		{ tag: tags.punctuation, color: 'var(--color-text-muted)' },
		{ tag: tags.labelName, color: 'var(--color-info)' },
		{ tag: tags.function(tags.variableName), color: 'var(--color-info)' },
		{ tag: tags.definition(tags.variableName), color: 'var(--color-text)' },
		{ tag: tags.propertyName, color: 'var(--color-info)' }
	]);
	import ResizeHandle from './ResizeHandle.svelte';
	import DataGrid from './DataGrid.svelte';
	import FKPreviewPopup from './FKPreviewPopup.svelte';
	import Icon from '$lib/icons/Icon.svelte';
	import { dataApi } from '$lib/api/client';
	import type { ColumnInfo, ForeignKeyPreview } from '$lib/types';

	// Error highlighting extension
	const setErrorEffect = StateEffect.define<{ from: number; to: number } | null>();

	const errorMark = Decoration.mark({ class: 'cm-error-highlight' });

	const errorHighlightField = StateField.define<DecorationSet>({
		create() {
			return Decoration.none;
		},
		update(decorations, tr) {
			for (const effect of tr.effects) {
				if (effect.is(setErrorEffect)) {
					if (effect.value === null) {
						return Decoration.none;
					}
					const builder = new RangeSetBuilder<Decoration>();
					builder.add(effect.value.from, effect.value.to, errorMark);
					return builder.finish();
				}
			}
			return decorations.map(tr.changes);
		},
		provide: (f) => EditorView.decorations.from(f)
	});

	const errorHighlightTheme = EditorView.theme({
		'.cm-error-highlight': {
			backgroundColor: '#f38ba8 !important',
			color: '#1e1e2e !important',
			borderRadius: '2px'
		}
	});

	let editorView: EditorView | null = null;
	let unsubscribeEditorActions: (() => void) | null = null;

	interface Props {
		tab: Tab;
		onSaveQuery?: (sql: string) => void;
	}

	let { tab, onSaveQuery }: Props = $props();

	// Track which tab ID we're currently showing
	let currentTabId = $state(tab.id);

	// Initialize query: use persisted content if available, otherwise use initialSql
	let query = $state(tab.queryContent ?? tab.initialSql ?? 'SELECT * FROM ');
	// Initialize result from tab.data if available (persisted across tab switches)
	let result = $state<QueryResult | null>((tab.data as QueryResult) ?? null);
	let isExecuting = $state(false);
	let executionTime = $state<number | null>(null);
	let containerEl: HTMLDivElement;

	// Pagination state for results
	let page = $state(1);
	let pageSize = $state(100);

	// Sorting state for results (client-side)
	let orderBy = $state<string | null>(null);
	let orderDir = $state<'ASC' | 'DESC'>('ASC');

	// FK preview state
	let fkPreview = $state<ForeignKeyPreview | null>(null);
	let fkPreviewPosition = $state({ x: 0, y: 0 });
	let fkPreviewLoading = $state(false);
	let hoverTimeout: ReturnType<typeof setTimeout> | null = null;
	let closeTimeout: ReturnType<typeof setTimeout> | null = null;

	// Derived: sorted rows
	let sortedRows = $derived.by(() => {
		if (!result || !orderBy) return result?.rows || [];
		const col = orderBy;
		const dir = orderDir;
		return [...result.rows].sort((a, b) => {
			const aVal = a[col];
			const bVal = b[col];
			// Handle nulls - null values go to the end
			if (aVal === null && bVal === null) return 0;
			if (aVal === null) return 1;
			if (bVal === null) return -1;
			// Compare values
			if (typeof aVal === 'number' && typeof bVal === 'number') {
				return dir === 'ASC' ? aVal - bVal : bVal - aVal;
			}
			const aStr = String(aVal);
			const bStr = String(bVal);
			return dir === 'ASC' ? aStr.localeCompare(bStr) : bStr.localeCompare(aStr);
		});
	});

	// Derived pagination values
	let totalPages = $derived(result ? Math.max(1, Math.ceil(result.rowCount / pageSize)) : 1);
	let paginatedRows = $derived.by(() => {
		if (!result) return [];
		const start = (page - 1) * pageSize;
		const end = start + pageSize;
		return sortedRows.slice(start, end);
	});

	// Detect tab switches and load the appropriate content
	// Use untrack for reading currentTabId to prevent infinite loop
	// (we're updating currentTabId inside the effect, so reading it would re-trigger)
	$effect(() => {
		const newTabId = tab.id;
		const oldTabId = untrack(() => currentTabId);

		if (newTabId !== oldTabId) {
			// Tab changed - load content for the new tab
			currentTabId = newTabId;
			query = tab.queryContent ?? tab.initialSql ?? 'SELECT * FROM ';
			// Load persisted result from tab.data
			result = (tab.data as QueryResult) ?? null;
			executionTime = result?.duration ?? null;
		}
	});

	// Persist query content to tab state when it changes
	// Use untrack for both the store update AND the tab.id read to prevent infinite loop:
	// Without untrack on tab.id, this effect re-runs when tabs.updateQueryContent updates
	// the tab, because the parent passes a new tab prop reference
	$effect(() => {
		const currentQuery = query;
		// Use untrack to avoid dependency on tab prop (prevents re-trigger loop)
		const tabId = untrack(() => currentTabId);
		// Only save if we have actual content
		if (currentQuery !== undefined && tabId) {
			untrack(() => {
				tabs.updateQueryContent(tabId, currentQuery);
			});
		}
	});

	// Persist query results to tab state when they change
	$effect(() => {
		const currentResult = result;
		const tabId = untrack(() => currentTabId);
		if (tabId) {
			untrack(() => {
				tabs.updateQueryResult(tabId, currentResult);
			});
		}
	});

	// Sync query changes to the editor store for Claude to access
	// Use untrack to prevent the store update from creating reactive dependencies
	$effect(() => {
		const currentQuery = query;
		untrack(() => {
			editorStore.setContent(currentQuery);
		});
	});

	// Subscribe to pending actions from Claude
	onMount(() => {
		unsubscribeEditorActions = editorStore.subscribeToPendingActions((action) => {
			if (action && editorView) {
				if (action.action === 'replace') {
					// Replace entire content
					query = action.text;
				} else if (action.action === 'insert') {
					// Insert at position or append
					if (action.position) {
						const doc = editorView.state.doc;
						const lines = doc.toJSON();
						let offset = 0;
						for (let i = 0; i < action.position.line && i < lines.length; i++) {
							offset += lines[i].length + 1;
						}
						offset += Math.min(action.position.column, lines[action.position.line]?.length || 0);

						const before = query.slice(0, offset);
						const after = query.slice(offset);
						query = before + action.text + after;
					} else {
						// Append to end
						query = query ? query + '\n' + action.text : action.text;
					}
				}
			}
		});
	});

	onDestroy(() => {
		if (unsubscribeEditorActions) {
			unsubscribeEditorActions();
		}
	});

	// Build reactive extensions with autocomplete based on schema data
	const extensions = $derived.by(() => {
		const schemaData = {
			tables: $tables || [],
			views: $views || [],
			functions: $functions || [],
			columns: $allColumns || []
		};

		const completionSource = createSchemaCompletionSource(schemaData);

		return [
			sql({ dialect: PostgreSQL }),
			appTheme,
			syntaxHighlighting(appHighlightStyle),
			autocompletion({
				override: [completionSource],
				activateOnTyping: true,
				maxRenderedOptions: 50
			}),
			errorHighlightField,
			errorHighlightTheme
		];
	});

	function highlightError(position: number) {
		if (!editorView) return;

		// Position is 1-based from PostgreSQL, convert to 0-based
		const pos = position - 1;
		const doc = editorView.state.doc.toString();
		const docLength = doc.length;

		// Clamp position to valid range
		const startPos = Math.max(0, Math.min(pos, docLength));

		// Find word boundaries around the error position
		let from = startPos;
		let to = startPos;

		// Expand backwards to find start of word
		while (from > 0 && /\w/.test(doc[from - 1])) {
			from--;
		}

		// Expand forwards to find end of word
		while (to < docLength && /\w/.test(doc[to])) {
			to++;
		}

		// If no word found, highlight at least a few characters
		if (from === to) {
			to = Math.min(startPos + 5, docLength);
		}

		editorView.dispatch({
			effects: setErrorEffect.of({ from, to })
		});
	}

	function clearErrorHighlight() {
		if (!editorView) return;
		editorView.dispatch({
			effects: setErrorEffect.of(null)
		});
	}

	function handleEditorReady(view: EditorView) {
		editorView = view;

		// Expose a test helper for E2E tests to set the query
		if (typeof window !== 'undefined') {
			(window as any).__PGVOYAGER_E2E__ = (window as any).__PGVOYAGER_E2E__ || {};
			(window as any).__PGVOYAGER_E2E__.setQuery = (newQuery: string) => {
				query = newQuery;
				// Also update CodeMirror directly
				if (view) {
					view.dispatch({
						changes: {
							from: 0,
							to: view.state.doc.length,
							insert: newQuery
						}
					});
				}
			};
		}
	}

	function handleEditorResize(delta: number) {
		if (!containerEl) return;
		const containerHeight = containerEl.offsetHeight;
		const deltaPercent = (delta / containerHeight) * 100;
		layout.setQueryEditorHeight($layout.queryEditorHeight + deltaPercent);
	}

	async function executeQuery() {
		if (!$activeConnectionId || !query.trim()) return;

		isExecuting = true;
		result = null;
		page = 1; // Reset pagination on new query
		orderBy = null; // Reset sorting on new query
		orderDir = 'ASC';
		clearErrorHighlight();

		const startTime = performance.now();

		try {
			const res = await queryApi.execute($activeConnectionId, query);
			result = res;
			executionTime = res.duration;

			// Check if the result contains an error (returned in response body)
			if (res.error) {
				// Highlight error position if available
				if (res.errorPosition && res.errorPosition > 0) {
					highlightError(res.errorPosition);
				}

				// Record failed query in history
				queryHistory.add({
					sql: query.trim(),
					connectionId: $activeConnectionId,
					connectionName: $activeConnection?.name || 'Unknown',
					duration: res.duration || performance.now() - startTime,
					rowCount: 0,
					success: false,
					error: res.error
				});
			} else {
				// Record successful query in history
				queryHistory.add({
					sql: query.trim(),
					connectionId: $activeConnectionId,
					connectionName: $activeConnection?.name || 'Unknown',
					duration: res.duration,
					rowCount: res.rowCount,
					success: true
				});
			}
		} catch (e) {
			const errorMessage = e instanceof Error ? e.message : 'Query failed';
			result = {
				columns: [],
				rows: [],
				rowCount: 0,
				duration: 0,
				error: errorMessage
			};

			// Record failed query in history
			queryHistory.add({
				sql: query.trim(),
				connectionId: $activeConnectionId,
				connectionName: $activeConnection?.name || 'Unknown',
				duration: performance.now() - startTime,
				rowCount: 0,
				success: false,
				error: errorMessage
			});
		} finally {
			isExecuting = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
			e.preventDefault();
			executeQuery();
		} else if ((e.ctrlKey || e.metaKey) && e.key === 's') {
			e.preventDefault();
			handleSave();
		}
	}

	function handleSave() {
		if (query.trim() && onSaveQuery) {
			onSaveQuery(query);
		}
	}

	function handleSort(column: string) {
		if (orderBy === column) {
			orderDir = orderDir === 'ASC' ? 'DESC' : 'ASC';
		} else {
			orderBy = column;
			orderDir = 'ASC';
		}
		page = 1; // Reset to first page when sorting
	}

	function handlePageChange(newPage: number) {
		page = newPage;
	}

	function handlePageSizeChange(newPageSize: number) {
		pageSize = newPageSize;
		page = 1;
	}

	// FK handlers
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

		// Update position immediately
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

	function formatCsvValue(value: unknown): string {
		if (value === null || value === undefined) return '';
		const str = typeof value === 'object' ? JSON.stringify(value) : String(value);
		// Escape quotes and wrap in quotes if contains comma, quote, or newline
		if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
			return `"${str.replace(/"/g, '""')}"`;
		}
		return str;
	}

	function exportToCsv() {
		if (!result || result.columns.length === 0) return;

		// Store in local const for type narrowing in callbacks
		const data = result;

		// Build CSV content
		const headers = data.columns.map(col => formatCsvValue(col.name)).join(',');
		const rows = data.rows.map(row =>
			data.columns.map(col => formatCsvValue(row[col.name])).join(',')
		);
		const csvContent = [headers, ...rows].join('\n');

		// Create and trigger download
		const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
		const url = URL.createObjectURL(blob);
		const link = document.createElement('a');
		link.href = url;
		link.download = `query_results_${new Date().toISOString().slice(0, 19).replace(/[:-]/g, '')}.csv`;
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
		URL.revokeObjectURL(url);
	}
</script>

<div class="query-editor" data-testid="query-editor" onkeydown={handleKeydown} bind:this={containerEl}>
	<div class="editor-section" style="flex: 0 0 {$layout.queryEditorHeight}%">
		<div class="editor-toolbar" data-testid="editor-toolbar">
			<button
				class="btn btn-primary btn-sm"
				data-testid="btn-run-query"
				onclick={executeQuery}
				disabled={isExecuting || !query.trim()}
			>
				{#if isExecuting}
					<Icon name="refresh" size={14} class="spinning" />
					Running...
				{:else}
					<Icon name="play" size={14} />
					Run (Ctrl+Enter)
				{/if}
			</button>
			<button
				class="btn btn-ghost btn-sm"
				data-testid="btn-save-query"
				onclick={handleSave}
				disabled={!query.trim()}
				title="Save Query (Ctrl+S)"
			>
				<Icon name="save" size={14} />
				Save
			</button>
			{#if executionTime !== null}
				<span class="execution-time" data-testid="execution-time">
					{executionTime.toFixed(2)}ms
				</span>
			{/if}
		</div>
		<div class="editor-container" data-testid="editor-container">
			<CodeMirror
				bind:value={query}
				{extensions}
				onready={handleEditorReady}
			/>
		</div>
	</div>

	<ResizeHandle direction="vertical" onResize={handleEditorResize} />

	<div class="results-section" data-testid="results-section">
		{#if isExecuting}
			<div class="results-loading" data-testid="results-loading">Executing query...</div>
		{:else if result?.error}
			<div class="results-error" data-testid="results-error">
				<h4>Error{#if result.errorPosition} at position {result.errorPosition}{/if}</h4>
				<pre>{result.error}</pre>
				{#if result.errorDetail}
					<div class="error-detail">
						<strong>Detail:</strong> {result.errorDetail}
					</div>
				{/if}
				{#if result.errorHint}
					<div class="error-hint">
						<strong>Hint:</strong> {result.errorHint}
					</div>
				{/if}
			</div>
		{:else if result}
			{#if result.columns.length > 0}
				<div class="results-header" data-testid="results-header">
					<span data-testid="row-count">{result.rowCount} row{result.rowCount !== 1 ? 's' : ''} returned</span>
				</div>
				<div class="results-grid-container" data-testid="results-table">
					<DataGrid
						columns={result.columns}
						rows={paginatedRows}
						totalRows={result.rowCount}
						{page}
						{pageSize}
						{totalPages}
						{orderBy}
						{orderDir}
						onSort={handleSort}
						onPageChange={handlePageChange}
						onPageSizeChange={handlePageSizeChange}
						showExport={true}
						onExport={exportToCsv}
						onFKClick={handleFKClick}
						onFKHover={handleFKHover}
						onFKLeave={handleFKLeave}
					/>
				</div>
			{:else}
				<div class="results-empty">Query executed successfully (no results)</div>
			{/if}
		{:else}
			<div class="results-empty">
				<p>Run a query to see results</p>
				<p class="hint">Press Ctrl+Enter to execute</p>
			</div>
		{/if}
	</div>
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

<style>
	.query-editor {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.editor-section {
		display: flex;
		flex-direction: column;
		min-height: 150px;
		overflow: hidden;
	}

	.editor-toolbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 12px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
	}

	.execution-time {
		font-size: 12px;
		color: var(--color-text-muted);
		font-family: var(--font-mono);
	}

	.editor-container {
		flex: 1;
		position: relative;
		min-height: 0;
		overflow: hidden;
	}

	.editor-container :global(.codemirror-wrapper) {
		position: absolute !important;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
	}

	.editor-container :global(.cm-editor) {
		position: absolute !important;
		top: 0 !important;
		left: 0 !important;
		right: 0 !important;
		bottom: 0 !important;
		height: auto !important;
		font-size: 14px;
	}

	.editor-container :global(.cm-scroller) {
		overflow: auto !important;
		scrollbar-width: auto !important;
		scrollbar-color: var(--color-text-dim) var(--color-bg-secondary) !important;
	}

	.editor-container :global(.cm-scroller)::-webkit-scrollbar {
		width: 12px !important;
		height: 12px !important;
		display: block !important;
	}

	.editor-container :global(.cm-scroller)::-webkit-scrollbar-track {
		background: var(--color-bg-secondary) !important;
	}

	.editor-container :global(.cm-scroller)::-webkit-scrollbar-thumb {
		background: var(--color-text-dim) !important;
		border-radius: 6px !important;
		border: 2px solid var(--color-bg-secondary) !important;
	}

	.editor-container :global(.cm-scroller)::-webkit-scrollbar-thumb:hover {
		background: var(--color-text-muted) !important;
	}

	.results-section {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.results-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.results-grid-container {
		flex: 1;
		overflow: hidden;
	}

	.results-loading,
	.results-empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		color: var(--color-text-muted);
	}

	.results-empty .hint {
		font-size: 12px;
		color: var(--color-text-dim);
		margin-top: 8px;
	}

	.results-error {
		padding: 16px;
		color: var(--color-error);
	}

	.results-error h4 {
		margin-bottom: 8px;
	}

	.results-error pre {
		background: var(--color-bg-tertiary);
		padding: 12px;
		border-radius: var(--radius-sm);
		overflow-x: auto;
		font-family: var(--font-mono);
		font-size: 12px;
		white-space: pre-wrap;
	}

	.error-detail,
	.error-hint {
		margin-top: 8px;
		padding: 8px 12px;
		background: var(--color-bg-tertiary);
		border-radius: var(--radius-sm);
		font-size: 12px;
	}

	.error-hint {
		color: var(--color-warning, #f9e2af);
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	/* Error highlight for SQL syntax errors */
	.editor-container :global(.cm-error-highlight) {
		background-color: #f38ba8 !important;
		color: #1e1e2e !important;
		border-radius: 2px;
	}

	/* Text selection highlight */
	.editor-container :global(.cm-selectionBackground) {
		background-color: rgba(137, 180, 250, 0.5) !important;
	}

</style>
