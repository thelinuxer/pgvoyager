<script lang="ts">
	import { onMount } from 'svelte';
	import { activeConnectionId } from '$lib/stores/connections';
	import { queryApi } from '$lib/api/client';
	import type { Tab, QueryResult } from '$lib/types';
	import CodeMirror from 'svelte-codemirror-editor';
	import { sql, PostgreSQL } from '@codemirror/lang-sql';
	import { oneDark } from '@codemirror/theme-one-dark';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let query = $state('SELECT * FROM ');
	let result = $state<QueryResult | null>(null);
	let isExecuting = $state(false);
	let executionTime = $state<number | null>(null);

	const extensions = [sql({ dialect: PostgreSQL }), oneDark];

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
</script>

<div class="query-editor" onkeydown={handleKeydown}>
	<div class="editor-section">
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
		height: 40%;
		min-height: 150px;
		border-bottom: 1px solid var(--color-border);
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
