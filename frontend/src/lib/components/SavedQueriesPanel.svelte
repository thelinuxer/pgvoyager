<script lang="ts">
	import { onMount } from 'svelte';
	import { savedQueries } from '$lib/stores/savedQueries';
	import { activeConnectionId, connections } from '$lib/stores/connections';
	import { tabs } from '$lib/stores/tabs';
	import type { SavedQuery } from '$lib/types';

	interface Props {
		onClose: () => void;
		onEditQuery?: (query: SavedQuery) => void;
	}

	let { onClose, onEditQuery }: Props = $props();

	let filterMode = $state<'all' | 'current'>('all');
	let searchQuery = $state('');
	let isLoading = $state(true);

	onMount(async () => {
		await savedQueries.load();
		isLoading = false;
	});

	// Get connection name by ID
	function getConnectionName(connectionId: string | undefined): string | null {
		if (!connectionId) return null;
		const conn = $connections.find((c) => c.id === connectionId);
		return conn?.name || 'Unknown';
	}

	// Filter queries based on mode and search
	let filteredQueries = $derived.by(() => {
		let queries = $savedQueries;

		// Filter by connection
		if (filterMode === 'current' && $activeConnectionId) {
			queries = queries.filter(
				(q) => !q.connectionId || q.connectionId === $activeConnectionId
			);
		}

		// Filter by search
		if (searchQuery.trim()) {
			const query = searchQuery.toLowerCase();
			queries = queries.filter(
				(q) =>
					q.name.toLowerCase().includes(query) ||
					q.sql.toLowerCase().includes(query) ||
					q.description?.toLowerCase().includes(query)
			);
		}

		// Sort by updatedAt descending
		return [...queries].sort(
			(a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
		);
	});

	function formatDate(isoString: string): string {
		const date = new Date(isoString);
		return date.toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	function truncateSql(sql: string, maxLength: number = 100): string {
		const oneLine = sql.replace(/\s+/g, ' ').trim();
		if (oneLine.length <= maxLength) return oneLine;
		return oneLine.substring(0, maxLength) + '...';
	}

	function handleQueryClick(query: SavedQuery) {
		tabs.openQuery({ title: query.name, initialSql: query.sql });
		onClose();
	}

	function handleEdit(e: MouseEvent, query: SavedQuery) {
		e.stopPropagation();
		onEditQuery?.(query);
	}

	async function handleDelete(e: MouseEvent, query: SavedQuery) {
		e.stopPropagation();
		if (!confirm(`Delete "${query.name}"?`)) return;

		try {
			await savedQueries.remove(query.id);
		} catch (err) {
			console.error('Failed to delete query:', err);
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="panel-backdrop" onclick={handleBackdropClick}>
	<div class="panel">
		<div class="panel-header">
			<h2>
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z"/>
					<polyline points="17 21 17 13 7 13 7 21"/>
					<polyline points="7 3 7 8 15 8"/>
				</svg>
				Saved Queries
			</h2>
			<button class="panel-close" onclick={onClose} title="Close">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M18 6L6 18M6 6l12 12"/>
				</svg>
			</button>
		</div>

		<div class="panel-toolbar">
			<div class="search-wrapper">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="11" cy="11" r="8"/>
					<path d="M21 21l-4.35-4.35"/>
				</svg>
				<input
					type="text"
					placeholder="Search saved queries..."
					bind:value={searchQuery}
				/>
			</div>
			<div class="filter-buttons">
				<button
					class="filter-btn"
					class:active={filterMode === 'current'}
					onclick={() => (filterMode = 'current')}
					disabled={!$activeConnectionId}
				>
					Current DB
				</button>
				<button
					class="filter-btn"
					class:active={filterMode === 'all'}
					onclick={() => (filterMode = 'all')}
				>
					All
				</button>
			</div>
		</div>

		<div class="panel-content">
			{#if isLoading}
				<div class="empty-state">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spinning">
						<path d="M23 4v6h-6M1 20v-6h6"/>
						<path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
					</svg>
					Loading...
				</div>
			{:else if filteredQueries.length === 0}
				<div class="empty-state">
					<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
						<path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z"/>
						<polyline points="17 21 17 13 7 13 7 21"/>
						<polyline points="7 3 7 8 15 8"/>
					</svg>
					<p>No saved queries</p>
					<span class="hint">Save queries from the query editor (Ctrl+S)</span>
				</div>
			{:else}
				<div class="queries-list">
					{#each filteredQueries as query}
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div
							class="query-item"
							onclick={() => handleQueryClick(query)}
							onkeydown={(e) => e.key === 'Enter' && handleQueryClick(query)}
							role="button"
							tabindex="0"
						>
							<div class="query-item-main">
								<div class="query-name">{query.name}</div>
								{#if query.description}
									<div class="query-description">{query.description}</div>
								{/if}
								<div class="query-sql">{truncateSql(query.sql)}</div>
								<div class="query-meta">
									<span class="query-date">{formatDate(query.updatedAt)}</span>
									{#if query.connectionId}
										<span class="query-db" title="Bound to specific connection">
											{getConnectionName(query.connectionId)}
										</span>
									{/if}
								</div>
							</div>
							<div class="query-actions">
								<button
									class="action-btn"
									onclick={(e) => handleEdit(e, query)}
									title="Edit"
								>
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
										<path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
									</svg>
								</button>
								<button
									class="action-btn danger"
									onclick={(e) => handleDelete(e, query)}
									title="Delete"
								>
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<polyline points="3 6 5 6 21 6"/>
										<path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
									</svg>
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div class="panel-footer">
			<span class="queries-count">{filteredQueries.length} saved queries</span>
		</div>
	</div>
</div>

<style>
	.panel-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.panel {
		background: var(--color-bg);
		border-radius: var(--radius-lg);
		box-shadow: 0 16px 64px rgba(0, 0, 0, 0.4);
		width: 100%;
		max-width: 650px;
		max-height: 80vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid var(--color-border);
	}

	.panel-header h2 {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 18px;
		font-weight: 600;
	}

	.panel-header h2 svg {
		color: var(--color-primary);
	}

	.panel-close {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		border-radius: var(--radius-sm);
		opacity: 0.5;
		transition: all var(--transition-fast);
	}

	.panel-close:hover {
		opacity: 1;
		background: var(--color-surface);
	}

	.panel-toolbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 20px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
	}

	.search-wrapper {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 12px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
	}

	.search-wrapper svg {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.search-wrapper input {
		flex: 1;
		border: none;
		background: transparent;
		font-size: 13px;
		padding: 0;
	}

	.search-wrapper input:focus {
		outline: none;
	}

	.filter-buttons {
		display: flex;
		gap: 4px;
	}

	.filter-btn {
		padding: 6px 12px;
		font-size: 12px;
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}

	.filter-btn:hover:not(:disabled) {
		background: var(--color-surface);
		color: var(--color-text);
	}

	.filter-btn.active {
		background: var(--color-primary);
		color: white;
	}

	.filter-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.panel-content {
		flex: 1;
		overflow-y: auto;
		min-height: 200px;
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
		color: var(--color-text-muted);
		text-align: center;
	}

	.empty-state svg {
		margin-bottom: 12px;
		opacity: 0.5;
	}

	.empty-state p {
		font-size: 14px;
	}

	.empty-state .hint {
		font-size: 12px;
		color: var(--color-text-dim);
		margin-top: 4px;
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.queries-list {
		display: flex;
		flex-direction: column;
	}

	.query-item {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px 20px;
		border-bottom: 1px solid var(--color-border);
		text-align: left;
		transition: background var(--transition-fast);
	}

	.query-item:hover {
		background: var(--color-surface);
	}

	.query-item:last-child {
		border-bottom: none;
	}

	.query-item-main {
		flex: 1;
		min-width: 0;
	}

	.query-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text);
		margin-bottom: 4px;
	}

	.query-description {
		font-size: 12px;
		color: var(--color-text-muted);
		margin-bottom: 6px;
	}

	.query-sql {
		font-family: var(--font-mono);
		font-size: 12px;
		color: var(--color-text-dim);
		margin-bottom: 8px;
		word-break: break-all;
	}

	.query-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 11px;
		color: var(--color-text-muted);
	}

	.query-db {
		padding: 2px 6px;
		background: var(--color-surface);
		border-radius: var(--radius-sm);
	}

	.query-actions {
		display: flex;
		gap: 4px;
		opacity: 0;
		transition: opacity var(--transition-fast);
	}

	.query-item:hover .query-actions {
		opacity: 1;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 6px;
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}

	.action-btn:hover {
		color: var(--color-text);
		background: var(--color-bg-tertiary);
	}

	.action-btn.danger:hover {
		color: var(--color-error);
		background: rgba(243, 139, 168, 0.1);
	}

	.panel-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 20px;
		border-top: 1px solid var(--color-border);
		background: var(--color-bg-secondary);
	}

	.queries-count {
		font-size: 12px;
		color: var(--color-text-muted);
	}
</style>
