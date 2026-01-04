<script lang="ts">
	import { activeConnection, activeConnectionId } from '$lib/stores/connections';
	import { schemaTree, expandedNodes, toggleNode, isLoading, error, refreshSchema } from '$lib/stores/schema';
	import { tabs } from '$lib/stores/tabs';
	import { dataApi } from '$lib/api/client';
	import type { SchemaTreeNode, Table } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		width: number;
		onNewConnection: () => void;
		onShowHistory?: () => void;
		onShowSavedQueries?: () => void;
	}

	let { width, onNewConnection, onShowHistory, onShowSavedQueries }: Props = $props();

	function handleShowHistory() {
		onShowHistory?.();
	}

	function handleShowSavedQueries() {
		onShowSavedQueries?.();
	}

	let searchQuery = $state('');

	// Context menu state
	let contextMenu = $state<{ node: SchemaTreeNode; x: number; y: number; menuType: 'table' | 'schema' } | null>(null);

	// Drop table confirmation modal state
	let dropTableModal = $state<{ schema: string; table: string; cascade: boolean } | null>(null);
	let isDropping = $state(false);
	let dropError = $state<string | null>(null);

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

	// Use $state + $effect instead of $derived with Svelte 4 store to prevent reactivity issues
	let filteredTree = $state<SchemaTreeNode[]>([]);

	$effect(() => {
		const tree = $schemaTree;
		const query = searchQuery;
		filteredTree = filterTree(tree, query);
	});

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
		if (node.type === 'table' && node.schema) {
			e.preventDefault();
			contextMenu = { node, x: e.clientX, y: e.clientY, menuType: 'table' };
		} else if (node.type === 'schema') {
			e.preventDefault();
			contextMenu = { node, x: e.clientX, y: e.clientY, menuType: 'schema' };
		}
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

	function handleViewTableERD(node: SchemaTreeNode) {
		if (!node.schema) return;
		tabs.openTableERD(node.schema, node.name);
		closeContextMenu();
	}

	function handleViewSchemaERD(node: SchemaTreeNode) {
		// For schema nodes, node.name is the schema name
		tabs.openSchemaERD(node.name);
		closeContextMenu();
	}

	function handleCopyName(node: SchemaTreeNode) {
		if (!node.schema) return;
		const fullName = `"${node.schema}"."${node.name}"`;
		navigator.clipboard.writeText(fullName);
		closeContextMenu();
	}

	function handleFilterTable(node: SchemaTreeNode) {
		if (!node.schema) return;
		const sql = `SELECT *
FROM "${node.schema}"."${node.name}"
WHERE column_name = 'value'
LIMIT 100;`;
		tabs.openQuery({ title: `Filter ${node.name}`, initialSql: sql });
		closeContextMenu();
	}

	function handleDropTableClick(node: SchemaTreeNode) {
		if (!node.schema) return;
		dropTableModal = { schema: node.schema, table: node.name, cascade: false };
		closeContextMenu();
	}

	function closeDropTableModal() {
		dropTableModal = null;
		dropError = null;
	}

	async function confirmDropTable() {
		if (!dropTableModal || !$activeConnectionId) return;

		isDropping = true;
		dropError = null;

		try {
			await dataApi.dropTable(
				$activeConnectionId,
				dropTableModal.schema,
				dropTableModal.table,
				dropTableModal.cascade
			);
			closeDropTableModal();
			// Refresh the schema tree
			refreshSchema();
		} catch (e) {
			dropError = e instanceof Error ? e.message : 'Failed to drop table';
		} finally {
			isDropping = false;
		}
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

{#snippet getIcon(node: SchemaTreeNode)}
	{#if node.type === 'schema'}
		<Icon name="folder" size={14} />
	{:else if node.type === 'folder'}
		<Icon name={isExpanded(node) ? 'folder-open' : 'folder'} size={14} />
	{:else if node.type === 'table'}
		<Icon name="table" size={14} />
	{:else if node.type === 'view'}
		<Icon name="eye" size={14} />
	{:else if node.type === 'function'}
		<Icon name="terminal" size={14} />
	{:else if node.type === 'sequence'}
		<Icon name="sequence" size={14} />
	{:else if node.type === 'type'}
		<Icon name="type" size={14} />
	{:else}
		<Icon name="file" size={14} />
	{/if}
{/snippet}

<aside class="sidebar" data-testid="sidebar" style="width: {width}px">
	<div class="sidebar-header">
		<span class="sidebar-title">
			<Icon name="search" size={12} />
			Explorer
		</span>
		<div class="sidebar-actions">
			{#if $activeConnection}
				<button
					class="btn btn-sm btn-ghost"
					onclick={refreshSchema}
					disabled={$isLoading}
					title="Refresh Schema"
					data-testid="btn-refresh-schema"
				>
					<Icon name="refresh" size={14} class={$isLoading ? 'spinning' : ''} />
				</button>
				<button class="btn btn-sm btn-ghost" onclick={handleShowSavedQueries} title="Saved Queries">
					<Icon name="save" size={14} />
				</button>
				<button class="btn btn-sm btn-ghost" onclick={handleShowHistory} title="Query History">
					<Icon name="clock" size={14} />
				</button>
				<button class="btn btn-sm btn-ghost" onclick={() => tabs.openAnalysis()} title="Analyze Database">
					<Icon name="activity" size={14} />
				</button>
				<button class="btn btn-sm btn-ghost" data-testid="btn-new-query" onclick={() => tabs.openQuery()} title="New Query">
					<Icon name="file-code" size={14} />
				</button>
			{/if}
		</div>
	</div>

	{#if $activeConnection && !$isLoading && !$error && $schemaTree.length > 0}
		<div class="search-container">
			<div class="search-input-wrapper">
				<Icon name="search" size={14} class="search-icon" />
				<input
					type="text"
					class="search-input"
					placeholder="Filter..."
					bind:value={searchQuery}
					onkeydown={handleSearchKeydown}
				/>
				{#if searchQuery}
					<button class="search-clear" onclick={clearSearch} title="Clear (Esc)">
						<Icon name="x" size={12} />
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
				<Icon name="layers" size={32} strokeWidth={1.5} />
				<p>No connection selected</p>
				<button class="btn btn-sm btn-secondary" onclick={onNewConnection}>
					<Icon name="plus" size={12} />
					Connect
				</button>
			</div>
		{:else if $isLoading}
			<div class="sidebar-loading">
				<Icon name="refresh" size={20} class="spinning" />
				Loading schema...
			</div>
		{:else if $error}
			<div class="sidebar-error">
				<Icon name="alert-circle" size={16} />
				{$error}
			</div>
		{:else if $schemaTree.length === 0}
			<div class="sidebar-empty">No schemas found</div>
		{:else}
			<div class="tree" data-testid="schema-tree">
				{#each filteredTree as node}
					{@render treeNode(node, 0)}
				{/each}
			</div>
		{/if}
	</div>
</aside>

<!-- Context Menu -->
{#if contextMenu}
	{@const menuNode = contextMenu.node}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="context-menu-backdrop" onclick={closeContextMenu}></div>
	<div class="context-menu" style="left: {contextMenu.x}px; top: {contextMenu.y}px">
		{#if contextMenu.menuType === 'table'}
			<button class="context-menu-item" onclick={() => handleShowFirst100(menuNode)}>
				<Icon name="arrow-up" size={14} />
				Show first 100 rows
			</button>
			<button class="context-menu-item" onclick={() => handleShowLast100(menuNode)}>
				<Icon name="arrow-down" size={14} />
				Show last 100 rows
			</button>
			<button class="context-menu-item" onclick={() => handleFilterTable(menuNode)}>
				<Icon name="filter" size={14} />
				Filter table...
			</button>
			<div class="context-menu-separator"></div>
			<button class="context-menu-item" onclick={() => handleViewTableERD(menuNode)}>
				<Icon name="share-2" size={14} />
				View ERD
			</button>
			<div class="context-menu-separator"></div>
			<button class="context-menu-item" onclick={() => handleOpenInQuery(menuNode)}>
				<Icon name="file" size={14} />
				Open in Query Editor
			</button>
			<button class="context-menu-item" onclick={() => handleCopyName(menuNode)}>
				<Icon name="copy" size={14} />
				Copy table name
			</button>
			<div class="context-menu-separator"></div>
			<button class="context-menu-item context-menu-item-danger" onclick={() => handleDropTableClick(menuNode)}>
				<Icon name="trash" size={14} />
				Drop table...
			</button>
		{:else if contextMenu.menuType === 'schema'}
			<button class="context-menu-item" onclick={() => handleViewSchemaERD(menuNode)}>
				<Icon name="share-2" size={14} />
				View Schema ERD
			</button>
		{/if}
	</div>
{/if}

<!-- Drop Table Confirmation Modal -->
{#if dropTableModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="modal-backdrop" onclick={closeDropTableModal}></div>
	<div class="modal">
		<div class="modal-header">
			<Icon name="alert-circle" size={20} />
			<h3>Drop Table</h3>
		</div>
		<div class="modal-body">
			<p>Are you sure you want to drop the table <strong>"{dropTableModal.schema}"."{dropTableModal.table}"</strong>?</p>
			<p class="warning-text">This action cannot be undone!</p>

			{#if dropError}
				<div class="modal-error">
					<Icon name="alert-circle" size={14} />
					{dropError}
				</div>
			{/if}

			<label class="cascade-option">
				<input type="checkbox" bind:checked={dropTableModal.cascade} />
				<span>CASCADE (also drop dependent objects)</span>
			</label>
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary btn-sm" onclick={closeDropTableModal} disabled={isDropping}>
				Cancel
			</button>
			<button class="btn btn-danger btn-sm" onclick={confirmDropTable} disabled={isDropping}>
				{#if isDropping}
					<Icon name="refresh" size={14} class="spinning" />
					Dropping...
				{:else}
					Drop Table
				{/if}
			</button>
		</div>
	</div>
{/if}

{#snippet treeNode(node: SchemaTreeNode, depth: number)}
	<div class="tree-item" data-testid="tree-item-{node.name}" data-node-type="{node.type}" style="padding-left: {depth * 16 + 8}px">
		<button
			class="tree-item-button"
			data-testid="tree-button-{node.name}"
			onclick={() => handleNodeClick(node)}
			ondblclick={() => handleDoubleClick(node)}
			oncontextmenu={(e) => handleContextMenu(e, node)}
		>
			{#if node.children && node.children.length > 0}
				<span class="tree-chevron" class:expanded={isExpanded(node)}>
					<Icon name="chevron-right" size={10} />
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
		{#if node.type === 'table' || node.type === 'schema'}
			<button
				class="tree-item-menu"
				onclick={(e) => {
					e.stopPropagation();
					const rect = e.currentTarget.getBoundingClientRect();
					const menuType = node.type === 'schema' ? 'schema' : 'table';
					contextMenu = { node, x: rect.right, y: rect.top, menuType };
				}}
				title="More options"
			>
				<Icon name="dots-vertical" size={14} />
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

	.context-menu-item-danger {
		color: var(--color-error);
	}

	.context-menu-item-danger:hover {
		background: rgba(243, 139, 168, 0.15);
	}

	.context-menu-item-danger svg {
		color: var(--color-error);
	}

	/* Modal styles */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		z-index: 1000;
	}

	.modal {
		position: fixed;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		z-index: 1001;
		min-width: 400px;
		max-width: 500px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
	}

	.modal-header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 16px 20px;
		border-bottom: 1px solid var(--color-border);
		color: var(--color-error);
	}

	.modal-header h3 {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
		color: var(--color-text);
	}

	.modal-body {
		padding: 20px;
	}

	.modal-body p {
		margin: 0 0 12px;
	}

	.warning-text {
		color: var(--color-error);
		font-weight: 500;
	}

	.modal-error {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		margin: 12px 0;
		background: rgba(243, 139, 168, 0.15);
		border: 1px solid var(--color-error);
		border-radius: var(--radius-sm);
		color: var(--color-error);
		font-size: 13px;
	}

	.cascade-option {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 16px;
		padding: 10px 12px;
		background: var(--color-surface);
		border-radius: var(--radius-sm);
		cursor: pointer;
	}

	.cascade-option input {
		width: 16px;
		height: 16px;
		accent-color: var(--color-primary);
	}

	.cascade-option span {
		font-size: 13px;
		color: var(--color-text-muted);
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 16px 20px;
		border-top: 1px solid var(--color-border);
	}
</style>
