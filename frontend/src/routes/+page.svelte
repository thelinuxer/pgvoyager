<script lang="ts">
	import Header from '$lib/components/Header.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TabBar from '$lib/components/TabBar.svelte';
	import ContentArea from '$lib/components/ContentArea.svelte';
	import ConnectionModal from '$lib/components/ConnectionModal.svelte';
	import QueryHistoryPanel from '$lib/components/QueryHistoryPanel.svelte';
	import SavedQueriesPanel from '$lib/components/SavedQueriesPanel.svelte';
	import SaveQueryModal from '$lib/components/SaveQueryModal.svelte';
	import SettingsModal from '$lib/components/SettingsModal.svelte';
	import ClaudeTerminalPanel from '$lib/components/ClaudeTerminalPanel.svelte';
	import ResizeHandle from '$lib/components/ResizeHandle.svelte';
	import { activeConnection } from '$lib/stores/connections';
	import { layout } from '$lib/stores/layout';
	import type { Connection, SavedQuery } from '$lib/types';
	import { onMount } from 'svelte';

	let showConnectionModal = $state(false);
	let showHistoryPanel = $state(false);
	let showSavedQueriesPanel = $state(false);
	let showSaveQueryModal = $state(false);
	let showSettingsModal = $state(false);
	let saveQuerySql = $state('');
	let editSavedQuery = $state<SavedQuery | null>(null);
	let editConnection = $state<Connection | null>(null);

	function handleNewConnection() {
		editConnection = null;
		showConnectionModal = true;
	}

	function handleEditConnection() {
		if ($activeConnection) {
			editConnection = $activeConnection;
			showConnectionModal = true;
		}
	}

	function handleCloseModal() {
		showConnectionModal = false;
		editConnection = null;
	}

	function handleShowHistory() {
		showHistoryPanel = true;
	}

	function handleCloseHistory() {
		showHistoryPanel = false;
	}

	function handleShowSavedQueries() {
		showSavedQueriesPanel = true;
	}

	function handleCloseSavedQueries() {
		showSavedQueriesPanel = false;
	}

	function handleSaveQuery(sql: string) {
		saveQuerySql = sql;
		editSavedQuery = null;
		showSaveQueryModal = true;
	}

	function handleEditSavedQuery(query: SavedQuery) {
		saveQuerySql = query.sql;
		editSavedQuery = query;
		showSavedQueriesPanel = false;
		showSaveQueryModal = true;
	}

	function handleCloseSaveQueryModal() {
		showSaveQueryModal = false;
		editSavedQuery = null;
	}

	function handleSettings() {
		showSettingsModal = true;
	}

	function handleCloseSettings() {
		showSettingsModal = false;
	}

	function handleSidebarResize(delta: number) {
		layout.setSidebarWidth($layout.sidebarWidth + delta);
	}

	function handleToggleClaude() {
		layout.toggleClaudeTerminal();
	}

	// Global keyboard shortcut for Claude terminal (Ctrl+`)
	function handleKeydown(e: KeyboardEvent) {
		if (e.ctrlKey && e.key === '`') {
			e.preventDefault();
			layout.toggleClaudeTerminal();
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});
</script>

<Header
	onNewConnection={handleNewConnection}
	onEditConnection={handleEditConnection}
	onSettings={handleSettings}
	onToggleClaude={handleToggleClaude}
/>

<div class="main-layout">
	<Sidebar
		width={$layout.sidebarWidth}
		onNewConnection={handleNewConnection}
		onShowHistory={handleShowHistory}
		onShowSavedQueries={handleShowSavedQueries}
	/>
	<ResizeHandle direction="horizontal" onResize={handleSidebarResize} />

	<div class="content-wrapper">
		<div class="content-main">
			{#if $activeConnection}
				<TabBar />
				<ContentArea onSaveQuery={handleSaveQuery} />
			{:else}
				<div class="welcome">
					<div class="welcome-content">
						<img src="/logo.svg" alt="PgVoyager" class="welcome-logo" />
						<h1>PgVoyager</h1>
						<p>Navigate your PostgreSQL databases with ease</p>
						<button class="btn btn-primary" onclick={handleNewConnection}>
							+ New Connection
						</button>
					</div>
				</div>
			{/if}
		</div>

		{#if $layout.claudeTerminalVisible && $activeConnection}
			<ClaudeTerminalPanel />
		{/if}
	</div>
</div>

{#if showConnectionModal}
	<ConnectionModal onClose={handleCloseModal} editConnection={editConnection} />
{/if}

{#if showHistoryPanel}
	<QueryHistoryPanel onClose={handleCloseHistory} />
{/if}

{#if showSavedQueriesPanel}
	<SavedQueriesPanel onClose={handleCloseSavedQueries} onEditQuery={handleEditSavedQuery} />
{/if}

{#if showSaveQueryModal}
	<SaveQueryModal sql={saveQuerySql} editQuery={editSavedQuery} onClose={handleCloseSaveQueryModal} />
{/if}

{#if showSettingsModal}
	<SettingsModal onClose={handleCloseSettings} />
{/if}

<style>
	.main-layout {
		flex: 1;
		display: flex;
		overflow: hidden;
	}

	.content-wrapper {
		flex: 1;
		display: flex;
		flex-direction: row;
		overflow: hidden;
		background: var(--color-bg);
	}

	.content-main {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.welcome {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.welcome-content {
		text-align: center;
	}

	.welcome-logo {
		width: 160px;
		height: auto;
		margin-bottom: 1rem;
	}

	.welcome-content h1 {
		font-size: 2.5rem;
		margin-bottom: 0.5rem;
	}

	.welcome-content p {
		color: var(--color-text-muted);
		margin-bottom: 2rem;
	}
</style>
