<script lang="ts">
	import { onMount } from 'svelte';
	import { activeConnectionId } from '$lib/stores/connections';
	import { queryApi } from '$lib/api/client';
	import { layout } from '$lib/stores/layout';
	import type { Tab, QueryResult } from '$lib/types';
	import CodeMirror from 'svelte-codemirror-editor';
	import { sql, PostgreSQL } from '@codemirror/lang-sql';
	import { oneDark } from '@codemirror/theme-one-dark';
	import ResizeHandle from './ResizeHandle.svelte';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let query = $state(tab.initialSql || 'SELECT * FROM ');
	let result = $state<QueryResult | null>(null);
	let isExecuting = $state(false);
	let executionTime = $state<number | null>(null);
	let containerEl: HTMLDivElement;

	const extensions = [sql({ dialect: PostgreSQL }), oneDark];

	// Update query if tab changes with new initialSql
	$effect(() => {
		if (tab.initialSql && query === 'SELECT * FROM ') {
			query = tab.initialSql;
		}
	});

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

		try {
			const res = await queryApi.execute($activeConnectionId, query);
			result = res;
			executionTime = res.duration;
		} catch (e) {
			result = {
				columns: [],
				rows: [],
				rowCount: 0,
				duration: 0,
				error: e instanceof Error ? e.message : 'Query failed'
			};
		} finally {
			isExecuting = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
			e.preventDefault();
			executeQuery();
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

		// Build CSV content
		const headers = result.columns.map(col => formatCsvValue(col.name)).join(',');
		const rows = result.rows.map(row =>
			result.columns.map(col => formatCsvValue(row[col.name])).join(',')
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

<div class="query-editor" onkeydown={handleKeydown} bind:this={containerEl}>
	<div class="editor-section" style="height: {$layout.queryEditorHeight}%">
		<div class="editor-toolbar">
			<button
				class="btn btn-primary btn-sm"
				onclick={executeQuery}
				disabled={isExecuting || !query.trim()}
			>
				{#if isExecuting}
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spinning">
						<path d="M23 4v6h-6M1 20v-6h6"/>
						<path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
					</svg>
					Running...
				{:else}
					<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
						<polygon points="5 3 19 12 5 21 5 3"/>
					</svg>
					Run (Ctrl+Enter)
				{/if}
			</button>
			{#if executionTime !== null}
				<span class="execution-time">
					{executionTime.toFixed(2)}ms
				</span>
			{/if}
		</div>
		<div class="editor-container">
			<CodeMirror
				bind:value={query}
				{extensions}
				styles={{
					'&': {
						height: '100%',
						fontSize: '14px'
					}
				}}
			/>
		</div>
	</div>

	<ResizeHandle direction="vertical" onResize={handleEditorResize} />

	<div class="results-section">
		{#if isExecuting}
			<div class="results-loading">Executing query...</div>
		{:else if result?.error}
			<div class="results-error">
				<h4>Error</h4>
				<pre>{result.error}</pre>
			</div>
		{:else if result}
			<div class="results-header">
				<span>{result.rowCount} row{result.rowCount !== 1 ? 's' : ''} returned</span>
				<button class="btn btn-sm btn-ghost" onclick={exportToCsv} title="Export to CSV">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
						<polyline points="7 10 12 15 17 10"/>
						<line x1="12" y1="15" x2="12" y2="3"/>
					</svg>
					Export CSV
				</button>
			</div>
			<div class="results-table-container">
				{#if result.columns.length > 0}
					<table class="data-table">
						<thead>
							<tr>
								{#each result.columns as col}
									<th>{col.name}</th>
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
		overflow: hidden;
	}

	.editor-container :global(.cm-editor) {
		height: 100%;
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

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
