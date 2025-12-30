<script lang="ts">
	import { tabs, activeTabId } from '$lib/stores/tabs';
	import type { Tab } from '$lib/types';

	function handleTabClick(tab: Tab) {
		activeTabId.set(tab.id);
	}

	function handleTabClose(e: MouseEvent, tab: Tab) {
		e.stopPropagation();
		tabs.close(tab.id);
	}

	function handleMiddleClick(e: MouseEvent, tab: Tab) {
		if (e.button === 1) {
			e.preventDefault();
			tabs.close(tab.id);
		}
	}

	function handleContextMenu(e: MouseEvent, tab: Tab) {
		e.preventDefault();
		// Could implement context menu here
	}

	function handleDoubleClick(tab: Tab) {
		tabs.togglePin(tab.id);
	}
</script>

{#snippet iconTable()}
	<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<rect x="3" y="3" width="18" height="18" rx="2"/>
		<path d="M3 9h18M3 15h18M9 3v18"/>
	</svg>
{/snippet}

{#snippet iconQuery()}
	<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
		<path d="M14 2v6h6"/>
		<path d="M10 12l-2 2 2 2M14 12l2 2-2 2"/>
	</svg>
{/snippet}

{#snippet iconView()}
	<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
		<circle cx="12" cy="12" r="3"/>
	</svg>
{/snippet}

{#snippet iconPin()}
	<svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1">
		<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"/>
		<circle cx="12" cy="9" r="2.5" fill="var(--color-bg)"/>
	</svg>
{/snippet}

{#snippet iconClose()}
	<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M18 6L6 18M6 6l12 12"/>
	</svg>
{/snippet}

{#snippet getTabIcon(tab: Tab)}
	{#if tab.type === 'table'}
		{@render iconTable()}
	{:else if tab.type === 'query'}
		{@render iconQuery()}
	{:else if tab.type === 'view'}
		{@render iconView()}
	{:else}
		{@render iconTable()}
	{/if}
{/snippet}

<div class="tab-bar">
	<div class="tabs-container">
		{#each $tabs as tab (tab.id)}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="tab"
				class:active={$activeTabId === tab.id}
				class:pinned={tab.isPinned}
				onclick={() => handleTabClick(tab)}
				onmousedown={(e) => handleMiddleClick(e, tab)}
				ondblclick={() => handleDoubleClick(tab)}
				oncontextmenu={(e) => handleContextMenu(e, tab)}
				title={tab.isPinned ? 'Double-click to unpin' : 'Double-click to pin'}
				role="tab"
				tabindex="0"
			>
				<span class="tab-icon">{@render getTabIcon(tab)}</span>
				<span class="tab-title">{tab.title}</span>
				{#if tab.isPinned}
					<span class="tab-pin" title="Pinned">{@render iconPin()}</span>
				{:else}
					<button
						class="tab-close"
						onclick={(e) => handleTabClose(e, tab)}
						title="Close"
					>
						{@render iconClose()}
					</button>
				{/if}
			</div>
		{/each}
	</div>

	<div class="tab-actions">
		<button
			class="btn btn-sm btn-ghost"
			onclick={() => tabs.openQuery()}
			title="New Query (Ctrl+N)"
		>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="12" y1="5" x2="12" y2="19"/>
				<line x1="5" y1="12" x2="19" y2="12"/>
			</svg>
		</button>
	</div>
</div>

<style>
	.tab-bar {
		display: flex;
		align-items: center;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
		min-height: 36px;
	}

	.tabs-container {
		display: flex;
		flex: 1;
		overflow-x: auto;
		scrollbar-width: none;
	}

	.tabs-container::-webkit-scrollbar {
		display: none;
	}

	.tab {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 12px;
		border-right: 1px solid var(--color-border);
		background: var(--color-bg-tertiary);
		white-space: nowrap;
		transition: background var(--transition-fast);
		max-width: 200px;
		cursor: pointer;
	}

	.tab:hover {
		background: var(--color-surface);
	}

	.tab.active {
		background: var(--color-bg);
		border-bottom: 2px solid var(--color-primary);
		margin-bottom: -1px;
	}

	.tab.pinned {
		background: var(--color-bg-secondary);
	}

	.tab.pinned.active {
		background: var(--color-bg);
	}

	.tab-icon {
		display: flex;
		align-items: center;
		color: var(--color-text-muted);
	}

	.tab-title {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		font-size: 13px;
	}

	.tab-pin {
		display: flex;
		align-items: center;
		color: var(--color-primary);
		opacity: 0.7;
	}

	.tab-close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		border-radius: var(--radius-sm);
		opacity: 0.5;
		transition: all var(--transition-fast);
	}

	.tab-close:hover {
		opacity: 1;
		background: var(--color-error);
		color: white;
	}

	.tab-actions {
		display: flex;
		padding: 0 8px;
	}
</style>
