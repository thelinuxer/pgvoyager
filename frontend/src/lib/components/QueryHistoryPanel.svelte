<script lang="ts">
	import { queryHistory, type QueryHistoryEntry } from '$lib/stores/queryHistory';
	import { activeConnectionId } from '$lib/stores/connections';
	import { tabs } from '$lib/stores/tabs';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		onClose: () => void;
	}

	let { onClose }: Props = $props();

	let filterMode = $state<'all' | 'current'>('current');
	let searchQuery = $state('');

	// Filter history based on mode and search
	let filteredHistory = $derived.by(() => {
		let entries = $queryHistory;

		// Filter by connection
		if (filterMode === 'current' && $activeConnectionId) {
			entries = entries.filter((e) => e.connectionId === $activeConnectionId);
		}

		// Filter by search
		if (searchQuery.trim()) {
			const query = searchQuery.toLowerCase();
			entries = entries.filter((e) => e.sql.toLowerCase().includes(query));
		}

		return entries;
	});

	function formatTime(isoString: string): string {
		const date = new Date(isoString);
		const now = new Date();
		const diff = now.getTime() - date.getTime();

		// Less than 1 minute
		if (diff < 60000) {
			return 'Just now';
		}

		// Less than 1 hour
		if (diff < 3600000) {
			const mins = Math.floor(diff / 60000);
			return `${mins}m ago`;
		}

		// Less than 24 hours
		if (diff < 86400000) {
			const hours = Math.floor(diff / 3600000);
			return `${hours}h ago`;
		}

		// Same year
		if (date.getFullYear() === now.getFullYear()) {
			return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
		}

		return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
	}

	function truncateSql(sql: string, maxLength: number = 80): string {
		const oneLine = sql.replace(/\s+/g, ' ').trim();
		if (oneLine.length <= maxLength) return oneLine;
		return oneLine.substring(0, maxLength) + '...';
	}

	function handleEntryClick(entry: QueryHistoryEntry) {
		tabs.openQuery({ title: 'Query', initialSql: entry.sql });
		onClose();
	}

	function handleDelete(e: MouseEvent, entry: QueryHistoryEntry) {
		e.stopPropagation();
		queryHistory.remove(entry.id);
	}

	function handleClearAll() {
		if (filterMode === 'current' && $activeConnectionId) {
			queryHistory.clearForConnection($activeConnectionId);
		} else {
			queryHistory.clear();
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
				<Icon name="clock" size={18} />
				Query History
			</h2>
			<button class="panel-close" onclick={onClose} title="Close">
				<Icon name="x" size={18} />
			</button>
		</div>

		<div class="panel-toolbar">
			<div class="search-wrapper">
				<Icon name="search" size={14} />
				<input
					type="text"
					placeholder="Search queries..."
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
			{#if filteredHistory.length === 0}
				<div class="empty-state">
					<Icon name="clock" size={32} strokeWidth={1.5} />
					<p>No query history</p>
					<span class="hint">Executed queries will appear here</span>
				</div>
			{:else}
				<div class="history-list">
					{#each filteredHistory as entry}
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div
							class="history-item"
							onclick={() => handleEntryClick(entry)}
							onkeydown={(e) => e.key === 'Enter' && handleEntryClick(entry)}
							role="button"
							tabindex="0"
						>
							<div class="history-item-main">
								<span class="history-sql">{truncateSql(entry.sql)}</span>
								<div class="history-meta">
									<span class="history-time">{formatTime(entry.executedAt)}</span>
									<span class="history-db" title={entry.connectionName}>{entry.connectionName}</span>
									{#if entry.success}
										<span class="history-rows">{entry.rowCount} rows</span>
										<span class="history-duration">{entry.duration.toFixed(1)}ms</span>
									{:else}
										<span class="history-error">Error</span>
									{/if}
								</div>
							</div>
							<button
								class="history-delete"
								onclick={(e) => handleDelete(e, entry)}
								title="Remove from history"
							>
								<Icon name="x" size={14} />
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		{#if filteredHistory.length > 0}
			<div class="panel-footer">
				<span class="history-count">{filteredHistory.length} queries</span>
				<button class="btn btn-sm btn-ghost" onclick={handleClearAll}>
					<Icon name="trash" size={14} />
					Clear {filterMode === 'current' ? 'Current' : 'All'}
				</button>
			</div>
		{/if}
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
		max-width: 600px;
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

	.history-list {
		display: flex;
		flex-direction: column;
	}

	.history-item {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 12px 20px;
		border-bottom: 1px solid var(--color-border);
		text-align: left;
		transition: background var(--transition-fast);
	}

	.history-item:hover {
		background: var(--color-surface);
	}

	.history-item:last-child {
		border-bottom: none;
	}

	.history-item-main {
		flex: 1;
		min-width: 0;
	}

	.history-sql {
		display: block;
		font-family: var(--font-mono);
		font-size: 13px;
		color: var(--color-text);
		margin-bottom: 6px;
		word-break: break-all;
	}

	.history-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 11px;
		color: var(--color-text-muted);
	}

	.history-time {
		color: var(--color-text-dim);
	}

	.history-db {
		max-width: 100px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.history-rows,
	.history-duration {
		color: var(--color-success);
	}

	.history-error {
		color: var(--color-error);
	}

	.history-delete {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		opacity: 0;
		transition: all var(--transition-fast);
	}

	.history-item:hover .history-delete {
		opacity: 1;
	}

	.history-delete:hover {
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

	.history-count {
		font-size: 12px;
		color: var(--color-text-muted);
	}
</style>
