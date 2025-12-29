<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { connections } from '$lib/stores/connections';

	let { children } = $props();

	onMount(() => {
		connections.load();
	});
</script>

<svelte:head>
	<title>PgVoyager</title>
</svelte:head>

<div class="app">
	{@render children()}
</div>

<style>
	.app {
		height: 100vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}
</style>
