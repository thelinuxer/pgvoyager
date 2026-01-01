<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
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
		'&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
			backgroundColor: 'rgba(137, 180, 250, 0.3)'
		},
		'.cm-activeLine': {
			backgroundColor: 'var(--color-surface)'
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
	import Icon from '$lib/icons/Icon.svelte';

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

	const errorHighlightTheme = EditorView.baseTheme({
		'.cm-error-highlight': {
			backgroundColor: 'rgba(255, 0, 0, 0.3)',
			borderBottom: '2px wavy #f38ba8'
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
	let result = $state<QueryResult | null>(null);
	let isExecuting = $state(false);
	let executionTime = $state<number | null>(null);
	let containerEl: HTMLDivElement;

	// Detect tab switches and load the appropriate content
	$effect(() => {
		if (tab.id !== currentTabId) {
			// Tab changed - load content for the new tab
			currentTabId = tab.id;
			query = tab.queryContent ?? tab.initialSql ?? 'SELECT * FROM ';
			result = null;
			executionTime = null;
		}
	});

	// Persist query content to tab state when it changes (debounced via effect batching)
	$effect(() => {
		// Only save if we have actual content and it's different from what's stored
		if (query !== undefined) {
			tabs.updateQueryContent(tab.id, query);
		}
	});

	// Sync query changes to the editor store for Claude to access
	$effect(() => {
		editorStore.setContent(query);
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
		const docLength = editorView.state.doc.length;

		// Clamp position to valid range
		const from = Math.max(0, Math.min(pos, docLength));
		// Highlight the character at the error position (or to end if at last position)
		const to = Math.min(from + 1, docLength);

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

	function formatValue(value: unknown): string {
		if (value === null) return 'NULL';
		if (value === undefined) return '';
		if (typeof value === 'object') {
			return JSON.stringify(value);
		}
		return String(value);
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
			<div class="results-header" data-testid="results-header">
				<span data-testid="row-count">{result.rowCount} row{result.rowCount !== 1 ? 's' : ''} returned</span>
				<button class="btn btn-sm btn-ghost" data-testid="btn-export-csv" onclick={exportToCsv} title="Export to CSV">
					<Icon name="download" size={14} />
					Export CSV
				</button>
			</div>
			<div class="results-table-container" data-testid="results-table">
				{#if result.columns.length > 0}
					<table class="data-table">
						<thead>
							<tr>
								{#each result.columns as col}
									<th data-column="{col.name}">{col.name}</th>
								{/each}
							</tr>
						</thead>
						<tbody>
							{#each result.rows as row}
								<tr>
									{#each result.columns as col}
										<td class:null-value={row[col.name] === null}>
											{formatValue(row[col.name])}
										</td>
									{/each}
								</tr>
							{/each}
						</tbody>
					</table>
				{:else}
					<div class="results-empty">Query executed successfully (no results)</div>
				{/if}
			</div>
		{:else}
			<div class="results-empty">
				<p>Run a query to see results</p>
				<p class="hint">Press Ctrl+Enter to execute</p>
			</div>
		{/if}
	</div>
</div>

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

	.results-table-container {
		flex: 1;
		overflow: auto;
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
</style>
