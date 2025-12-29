<script lang="ts">
	import { activeTab } from '$lib/stores/tabs';
	import TableViewer from './TableViewer.svelte';
	import QueryEditor from './QueryEditor.svelte';
	import ViewViewer from './ViewViewer.svelte';
</script>

<div class="content-area">
	{#if $activeTab}
		{#if $activeTab.type === 'table'}
			<TableViewer tab={$activeTab} />
		{:else if $activeTab.type === 'query'}
			<QueryEditor tab={$activeTab} />
		{:else if $activeTab.type === 'view'}
			<ViewViewer tab={$activeTab} />
		{/if}
	{:else}
		<div class="no-tab">
			<p>No tab selected</p>
			<p class="hint">Select a table from the sidebar or open a new query</p>
		</div>
	{/if}
</div>

<style>
	.content-area {
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.no-tab {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		color: var(--color-text-muted);
	}

	.hint {
		font-size: 12px;
		color: var(--color-text-dim);
	}
</style>
