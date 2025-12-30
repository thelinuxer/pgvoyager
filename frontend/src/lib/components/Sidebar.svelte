<script lang="ts">
	import { activeConnection } from '$lib/stores/connections';
	import { schemaTree, expandedNodes, toggleNode, isLoading, error } from '$lib/stores/schema';
	import { tabs } from '$lib/stores/tabs';
	import type { SchemaTreeNode, Table } from '$lib/types';

	interface Props {
		width: number;
		onNewConnection: () => void;
	}

	let { width, onNewConnection }: Props = $props();

	let searchQuery = $state('');

	// Context menu state
	let contextMenu = $state<{ node: SchemaTreeNode; x: number; y: number } | null>(null);

	// Filter the tree based on search query
	function filterTree(nodes: SchemaTreeNode[], query: string): SchemaTreeNode[] {
		if (!query.trim()) return nodes;

		const lowerQuery = query.toLowerCase();

		function nodeMatches(node: SchemaTreeNode): boolean {
			// Check if node name matches
			if (node.name.toLowerCase().includes(lowerQuery)) return true;

			// Check if any children match
			if (node.children) {
				return node.children.some(child => nodeMatches(child));
			}

			return false;
		}

		function filterNode(node: SchemaTreeNode): SchemaTreeNode | null {
			// If this node directly matches, include it with filtered children
			const nameMatches = node.name.toLowerCase().includes(lowerQuery);

			if (node.children) {
				const filteredChildren = node.children
					.map(child => filterNode(child))
					.filter((child): child is SchemaTreeNode => child !== null);

				// Include if name matches OR has matching children
				if (nameMatches || filteredChildren.length > 0) {
					return {
						...node,
						children: nameMatches ? node.children : filteredChildren
					};
				}
			} else if (nameMatches) {
				return node;
			}

			return null;
		}

		return nodes
			.map(node => filterNode(node))
			.filter((node): node is SchemaTreeNode => node !== null);
	}

	let filteredTree = $derived(filterTree($schemaTree, searchQuery));

	function handleNodeClick(node: SchemaTreeNode) {
		if (node.type === 'schema' || node.type === 'folder') {
			const key = node.schema ? `${node.schema}:${node.name}` : node.name;
			toggleNode(key);
		} else if (node.type === 'table' && node.schema) {
			tabs.openTable(node.schema, node.name);
		} else if (node.type === 'view' && node.schema) {
			tabs.openView(node.schema, node.name);
		} else if (node.type === 'function' && node.schema) {
			tabs.openFunction(node.schema, node.name);
		} else if (node.type === 'sequence' && node.schema) {
			tabs.openSequence(node.schema, node.name);
		} else if (node.type === 'type' && node.schema) {
			tabs.openType(node.schema, node.name);
		}
	}

	function handleDoubleClick(node: SchemaTreeNode) {
		if (node.type === 'table' && node.schema) {
			tabs.openTable(node.schema, node.name);
		} else if (node.type === 'view' && node.schema) {
			tabs.openView(node.schema, node.name);
		} else if (node.type === 'function' && node.schema) {
			tabs.openFunction(node.schema, node.name);
		} else if (node.type === 'sequence' && node.schema) {
			tabs.openSequence(node.schema, node.name);
		} else if (node.type === 'type' && node.schema) {
			tabs.openType(node.schema, node.name);
		}
	}

	function isExpanded(node: SchemaTreeNode): boolean {
		// When searching, expand all nodes to show matches
		if (searchQuery.trim()) return true;

		const key = node.schema ? `${node.schema}:${node.name}` : node.name;
		return $expandedNodes.has(key);
	}

	function handleContextMenu(e: MouseEvent, node: SchemaTreeNode) {
		if (node.type !== 'table' || !node.schema) return;
		e.preventDefault();
		contextMenu = { node, x: e.clientX, y: e.clientY };
	}

	function closeContextMenu() {
		contextMenu = null;
	}

	function getPrimaryKeyColumn(node: SchemaTreeNode): string | null {
		// Try to get primary key from table data if available
		// For now, default to 'id' as a common convention
		return 'id';
	}

	function handleShowFirst100(node: SchemaTreeNode) {
		if (!node.schema) return;
		const pkColumn = getPrimaryKeyColumn(node);
		tabs.openTable(node.schema, node.name, {
			sort: pkColumn ? { column: pkColumn, direction: 'ASC' } : undefined,
			limit: 100,
			forceNew: true
		});
		closeContextMenu();
	}

	function handleShowLast100(node: SchemaTreeNode) {
		if (!node.schema) return;
		const pkColumn = getPrimaryKeyColumn(node);
		tabs.openTable(node.schema, node.name, {
			sort: pkColumn ? { column: pkColumn, direction: 'DESC' } : undefined,
			limit: 100,
			forceNew: true
		});
		closeContextMenu();
	}

	function handleOpenInQuery(node: SchemaTreeNode) {
		if (!node.schema) return;
		const sql = `SELECT *\nFROM "${node.schema}"."${node.name}"\nLIMIT 100;`;
		tabs.openQuery({ title: `${node.schema}.${node.name}`, initialSql: sql });
		closeContextMenu();
	}

	function handleCopyName(node: SchemaTreeNode) {
		if (!node.schema) return;
		const fullName = `"${node.schema}"."${node.name}"`;
		navigator.clipboard.writeText(fullName);
		closeContextMenu();
	}

	function clearSearch() {
		searchQuery = '';
	}

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			clearSearch();
		}
	}
</script>

{#snippet iconSchema()}
	<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/>
	</svg>
{/snippet}

{#snippet iconFolder(expanded: boolean)}
	{#if expanded}
		<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/>
			<path d="M2 10h20"/>
		</svg>
	{:else}
		<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/>
		</svg>
	{/if}
{/snippet}

{#snippet iconTable()}
	<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<rect x="3" y="3" width="18" height="18" rx="2"/>
		<path d="M3 9h18M3 15h18M9 3v18"/>
	</svg>
{/snippet}

{#snippet iconView()}
	<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
		<circle cx="12" cy="12" r="3"/>
	</svg>
{/snippet}

{#snippet iconFunction()}
	<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M4 17l6-6-6-6M12 19h8"/>
	</svg>
{/snippet}

{#snippet iconSequence()}
	<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M12 2v20M2 12h20"/>
		<path d="M12 2l4 4-4 4"/>
	</svg>
{/snippet}

{#snippet iconType()}
	<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
		<path d="M4 7V4h16v3"/>
		<path d="M9 20h6"/>
		<path d="M12 4v16"/>
	</svg>
{/snippet}

{#snippet getIcon(node: SchemaTreeNode)}
	{#if node.type === 'schema'}
		{@render iconSchema()}
	{:else if node.type === 'folder'}
		{@render iconFolder(isExpanded(node))}
	{:else if node.type === 'table'}
		{@render iconTable()}
	{:else if node.type === 'view'}
		{@render iconView()}
	{:else if node.type === 'function'}
		{@render iconFunction()}
	{:else if node.type === 'sequence'}
		{@render iconSequence()}
	{:else if node.type === 'type'}
		{@render iconType()}
	{:else}
		<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
			<path d="M14 2v6h6M16 13H8M16 17H8M10 9H8"/>
		</svg>
	{/if}
{/snippet}

<aside class="sidebar" style="width: {width}px">
	<div class="sidebar-header">
		<span class="sidebar-title">
			<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="11" cy="11" r="8"/>
				<path d="M21 21l-4.35-4.35"/>
			</svg>
			Explorer
		</span>
		<div class="sidebar-actions">
			<button class="btn btn-sm btn-ghost" onclick={() => tabs.openQuery()} title="New Query">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
					<path d="M14 2v6h6"/>
					<line x1="12" y1="18" x2="12" y2="12"/>
					<line x1="9" y1="15" x2="15" y2="15"/>
				</svg>
			</button>
		</div>
	</div>

	{#if $activeConnection && !$isLoading && !$error && $schemaTree.length > 0}
		<div class="search-container">
			<div class="search-input-wrapper">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="search-icon">
					<circle cx="11" cy="11" r="8"/>
					<path d="M21 21l-4.35-4.35"/>
				</svg>
				<input
					type="text"
					class="search-input"
					placeholder="Filter..."
					bind:value={searchQuery}
					onkeydown={handleSearchKeydown}
				/>
				{#if searchQuery}
					<button class="search-clear" onclick={clearSearch} title="Clear (Esc)">
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M18 6L6 18M6 6l12 12"/>
						</svg>
					</button>
				{/if}
			</div>
			{#if searchQuery && filteredTree.length === 0}
				<div class="no-results">No matches found</div>
			{/if}
		</div>
	{/if}

	<div class="sidebar-content">
		{#if !$activeConnection}
			<div class="sidebar-empty">
				<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M12 2L2 7l10 5 10-5-10-5z"/>
					<path d="M2 17l10 5 10-5"/>
					<path d="M2 12l10 5 10-5"/>
				</svg>
				<p>No connection selected</p>
				<button class="btn btn-sm btn-secondary" onclick={onNewConnection}>
					<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="12" y1="5" x2="12" y2="19"/>
						<line x1="5" y1="12" x2="19" y2="12"/>
					</svg>
					Connect
				</button>
			</div>
		{:else if $isLoading}
			<div class="sidebar-loading">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spinning">
					<path d="M23 4v6h-6M1 20v-6h6"/>
					<path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
				</svg>
				Loading schema...
			</div>
		{:else if $error}
			<div class="sidebar-error">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10"/>
					<path d="M12 8v4M12 16h.01"/>
				</svg>
				{$error}
			</div>
		{:else if $schemaTree.length === 0}
			<div class="sidebar-empty">No schemas found</div>
		{:else}
			<div class="tree">
				{#each filteredTree as node}
					{@render treeNode(node, 0)}
				{/each}
			</div>
		{/if}
	</div>
</aside>

<!-- Context Menu -->
{#if contextMenu}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="context-menu-backdrop" onclick={closeContextMenu}></div>
	<div class="context-menu" style="left: {contextMenu.x}px; top: {contextMenu.y}px">
		<button class="context-menu-item" onclick={() => handleShowFirst100(contextMenu.node)}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 19V5M5 12l7-7 7 7"/>
			</svg>
			Show first 100 rows
		</button>
		<button class="context-menu-item" onclick={() => handleShowLast100(contextMenu.node)}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 5v14M5 12l7 7 7-7"/>
			</svg>
			Show last 100 rows
		</button>
		<div class="context-menu-separator"></div>
		<button class="context-menu-item" onclick={() => handleOpenInQuery(contextMenu.node)}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
				<path d="M14 2v6h6"/>
			</svg>
			Open in Query Editor
		</button>
		<button class="context-menu-item" onclick={() => handleCopyName(contextMenu.node)}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="9" y="9" width="13" height="13" rx="2"/>
				<path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/>
			</svg>
			Copy table name
		</button>
	</div>
{/if}

{#snippet treeNode(node: SchemaTreeNode, depth: number)}
	<div class="tree-item" style="padding-left: {depth * 16 + 8}px">
		<button
			class="tree-item-button"
			onclick={() => handleNodeClick(node)}
			ondblclick={() => handleDoubleClick(node)}
			oncontextmenu={(e) => handleContextMenu(e, node)}
		>
			{#if node.children && node.children.length > 0}
				<span class="tree-chevron" class:expanded={isExpanded(node)}>
					<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M9 18l6-6-6-6"/>
					</svg>
				</span>
			{:else}
				<span class="tree-spacer"></span>
			{/if}
			<span class="tree-icon">{@render getIcon(node)}</span>
			<span class="tree-label">{node.name}</span>
			{#if node.type === 'table' && node.data && 'rowCount' in node.data}
				<span class="tree-badge">{node.data.rowCount.toLocaleString()}</span>
			{/if}
		</button>
		{#if node.type === 'table'}
			<button
				class="tree-item-menu"
				onclick={(e) => {
					e.stopPropagation();
					const rect = e.currentTarget.getBoundingClientRect();
					contextMenu = { node, x: rect.right, y: rect.top };
				}}
				title="More options"
			>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="1"/>
					<circle cx="12" cy="5" r="1"/>
					<circle cx="12" cy="19" r="1"/>
				</svg>
			</button>
		{/if}
	</div>

	{#if node.children && isExpanded(node)}
		{#each node.children as child}
			{@render treeNode(child, depth + 1)}
		{/each}
	{/if}
{/snippet}

<style>
	.sidebar {
		display: flex;
		flex-direction: column;
		background: var(--color-bg-secondary);
		border-right: 1px solid var(--color-border);
		min-width: 200px;
		max-width: 500px;
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		border-bottom: 1px solid var(--color-border);
	}

	.sidebar-title {
		display: flex;
		align-items: center;
		gap: 6px;
		font-weight: 600;
		font-size: 12px;
		text-transform: uppercase;
		color: var(--color-text-muted);
	}

	.sidebar-actions {
		display: flex;
		gap: 4px;
	}

	.search-container {
		padding: 8px 12px;
		border-bottom: 1px solid var(--color-border);
	}

	.search-input-wrapper {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 10px;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		transition: border-color var(--transition-fast);
	}

	.search-input-wrapper:focus-within {
		border-color: var(--color-primary);
	}

	.search-icon {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.search-input {
		flex: 1;
		border: none;
		background: transparent;
		font-size: 13px;
		padding: 0;
		min-width: 0;
	}

	.search-input:focus {
		outline: none;
	}

	.search-input::placeholder {
		color: var(--color-text-dim);
	}

	.search-clear {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2px;
		border-radius: 2px;
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}

	.search-clear:hover {
		color: var(--color-text);
		background: var(--color-bg-tertiary);
	}

	.no-results {
		padding: 8px 0 0;
		font-size: 12px;
		color: var(--color-text-muted);
		text-align: center;
	}

	.sidebar-content {
		flex: 1;
		overflow-y: auto;
		padding: 8px 0;
	}

	.sidebar-empty,
	.sidebar-loading,
	.sidebar-error {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 24px;
		text-align: center;
		color: var(--color-text-muted);
		gap: 12px;
	}

	.sidebar-error {
		color: var(--color-error);
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.tree {
		font-size: 13px;
	}

	.tree-item {
		display: flex;
	}

	.tree-item-button {
		display: flex;
		align-items: center;
		gap: 4px;
		width: 100%;
		padding: 4px 8px;
		text-align: left;
		border-radius: var(--radius-sm);
		transition: background var(--transition-fast);
	}

	.tree-item-button:hover {
		background: var(--color-surface);
	}

	.tree-chevron {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		color: var(--color-text-muted);
		transition: transform var(--transition-fast);
	}

	.tree-chevron.expanded {
		transform: rotate(90deg);
	}

	.tree-spacer {
		width: 16px;
	}

	.tree-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		color: var(--color-text-muted);
	}

	.tree-label {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.tree-badge {
		font-size: 10px;
		color: var(--color-text-dim);
		padding: 1px 6px;
		background: var(--color-surface);
		border-radius: 8px;
	}

	.tree-item-menu {
		display: none;
		align-items: center;
		justify-content: center;
		padding: 2px;
		margin-right: 4px;
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}

	.tree-item:hover .tree-item-menu {
		display: flex;
	}

	.tree-item-menu:hover {
		color: var(--color-text);
		background: var(--color-bg-tertiary);
	}

	/* Context Menu */
	.context-menu-backdrop {
		position: fixed;
		inset: 0;
		z-index: 999;
	}

	.context-menu {
		position: fixed;
		z-index: 1000;
		min-width: 180px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
		padding: 4px;
	}

	.context-menu-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 8px 12px;
		font-size: 13px;
		text-align: left;
		border-radius: var(--radius-sm);
		transition: background var(--transition-fast);
	}

	.context-menu-item:hover {
		background: var(--color-surface);
	}

	.context-menu-item svg {
		color: var(--color-text-muted);
	}

	.context-menu-separator {
		height: 1px;
		background: var(--color-border);
		margin: 4px 0;
	}
</style>
