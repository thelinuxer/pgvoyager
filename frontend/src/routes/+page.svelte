<script lang="ts">
	import Header from '$lib/components/Header.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TabBar from '$lib/components/TabBar.svelte';
	import ContentArea from '$lib/components/ContentArea.svelte';
	import ConnectionModal from '$lib/components/ConnectionModal.svelte';
	import { activeConnection } from '$lib/stores/connections';

	let showConnectionModal = $state(false);
	let sidebarWidth = $state(280);
</script>

<Header onNewConnection={() => (showConnectionModal = true)} />

<div class="main-layout">
	<Sidebar width={sidebarWidth} onNewConnection={() => (showConnectionModal = true)} />

	<div class="content-wrapper">
		{#if $activeConnection}
			<TabBar />
			<ContentArea />
		{:else}
			<div class="welcome">
				<div class="welcome-content">
					<h1>🚀 PgVoyager</h1>
					<p>Navigate your PostgreSQL databases with ease</p>
					<button class="btn btn-primary" onclick={() => (showConnectionModal = true)}>
						+ New Connection
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>

{#if showConnectionModal}
	<ConnectionModal onClose={() => (showConnectionModal = false)} />
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
