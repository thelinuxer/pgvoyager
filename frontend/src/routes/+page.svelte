<script lang="ts">
	import Header from '$lib/components/Header.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TabBar from '$lib/components/TabBar.svelte';
	import ContentArea from '$lib/components/ContentArea.svelte';
	import ConnectionModal from '$lib/components/ConnectionModal.svelte';
	import ResizeHandle from '$lib/components/ResizeHandle.svelte';
	import { activeConnection } from '$lib/stores/connections';
	import { layout } from '$lib/stores/layout';
	import type { Connection } from '$lib/types';

	let showConnectionModal = $state(false);
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

	function handleSidebarResize(delta: number) {
		layout.setSidebarWidth($layout.sidebarWidth + delta);
	}
</script>

<Header onNewConnection={handleNewConnection} onEditConnection={handleEditConnection} />

<div class="main-layout">
	<Sidebar width={$layout.sidebarWidth} onNewConnection={handleNewConnection} />
	<ResizeHandle direction="horizontal" onResize={handleSidebarResize} />

	<div class="content-wrapper">
		{#if $activeConnection}
			<TabBar />
			<ContentArea />
		{:else}
			<div class="welcome">
				<div class="welcome-content">
					<h1>🚀 PgVoyager</h1>
					<p>Navigate your PostgreSQL databases with ease</p>
					<button class="btn btn-primary" onclick={handleNewConnection}>
						+ New Connection
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>

{#if showConnectionModal}
	<ConnectionModal onClose={handleCloseModal} editConnection={editConnection} />
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
		flex-direction: column;
		overflow: hidden;
		background: var(--color-bg);
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

	.welcome-content h1 {
		font-size: 2.5rem;
		margin-bottom: 0.5rem;
	}

	.welcome-content p {
		color: var(--color-text-muted);
		margin-bottom: 2rem;
	}
</style>
