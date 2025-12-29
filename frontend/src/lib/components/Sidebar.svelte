<script lang="ts">
	import { activeConnection } from '$lib/stores/connections';
	import { schemaTree, expandedNodes, toggleNode, isLoading, error } from '$lib/stores/schema';
	import { tabs } from '$lib/stores/tabs';
	import type { SchemaTreeNode } from '$lib/types';

	interface Props {
		width: number;
		onNewConnection: () => void;
	}

	let { width, onNewConnection }: Props = $props();

	function handleNodeClick(node: SchemaTreeNode) {
		if (node.type === 'schema' || node.type === 'folder') {
			const key = node.schema ? `${node.schema}:${node.name}` : node.name;
			toggleNode(key);
		} else if (node.type === 'table' && node.schema) {
			tabs.openTable(node.schema, node.name);
		} else if (node.type === 'view' && node.schema) {
			tabs.openView(node.schema, node.name);
		}
	}

	function handleDoubleClick(node: SchemaTreeNode) {
		if (node.type === 'table' && node.schema) {
			tabs.openTable(node.schema, node.name);
		} else if (node.type === 'view' && node.schema) {
			tabs.openView(node.schema, node.name);
		}
	}

	function isExpanded(node: SchemaTreeNode): boolean {
		const key = node.schema ? `${node.schema}:${node.name}` : node.name;
		return $expandedNodes.has(key);
	}

	function getIcon(node: SchemaTreeNode): string {
		switch (node.type) {
			case 'schema':
				return '📁';
			case 'folder':
				return isExpanded(node) ? '📂' : '📁';
			case 'table':
				return '📋';
			case 'view':
				return '👁';
			case 'function':
				return 'ƒ';
			case 'sequence':
				return '🔢';
			case 'type':
				return '🏷';
			default:
				return '📄';
		}
	}
</script>

<aside class="sidebar" style="width: {width}px">
	<div class="sidebar-header">
		<span class="sidebar-title">Explorer</span>
		<div class="sidebar-actions">
			<button class="btn btn-sm btn-ghost" onclick={() => tabs.openQuery()} title="New Query">
				📝
			</button>
		</div>
	</div>

	<div class="sidebar-content">
		{#if !$activeConnection}
			<div class="sidebar-empty">
				<p>No connection selected</p>
				<button class="btn btn-sm btn-secondary" onclick={onNewConnection}>
					+ Connect
				</button>
			</div>
		{:else if $isLoading}
			<div class="sidebar-loading">Loading schema...</div>
		{:else if $error}
			<div class="sidebar-error">{$error}</div>
		{:else if $schemaTree.length === 0}
			<div class="sidebar-empty">No schemas found</div>
		{:else}
			<div class="tree">
				{#each $schemaTree as node}
					{@render treeNode(node, 0)}
				{/each}
			</div>
		{/if}
	</div>
</aside>

{#snippet treeNode(node: SchemaTreeNode, depth: number)}
	<div class="tree-item" style="padding-left: {depth * 16 + 8}px">
		<button
			class="tree-item-button"
			onclick={() => handleNodeClick(node)}
			ondblclick={() => handleDoubleClick(node)}
		>
			{#if node.children && node.children.length > 0}
				<span class="tree-chevron" class:expanded={isExpanded(node)}>▶</span>
			{:else}
				<span class="tree-spacer"></span>
			{/if}
			<span class="tree-icon">{getIcon(node)}</span>
			<span class="tree-label">{node.name}</span>
			{#if node.type === 'table' && node.data && 'rowCount' in node.data}
				<span class="tree-badge">{node.data.rowCount.toLocaleString()}</span>
			{/if}
		</button>
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
		font-weight: 600;
		font-size: 12px;
		text-transform: uppercase;
		color: var(--color-text-muted);
	}

	.sidebar-actions {
		display: flex;
		gap: 4px;
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
		width: 16px;
		font-size: 10px;
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
		width: 16px;
		text-align: center;
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
</style>
