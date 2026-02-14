<script lang="ts">
	import { activeConnection, activeConnectionId } from '$lib/stores/connections';
	import { schemaTree, expandedNodes, toggleNode, isLoading, error, refreshSchema, loadTableColumns } from '$lib/stores/schema';
	import { tabs } from '$lib/stores/tabs';
	import { dataApi } from '$lib/api/client';
	import type { SchemaTreeNode, Table, Column } from '$lib/types';
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

	// Filter table modal state
	interface FilterCondition {
		column: string;
		operator: string;
		value: string;
	}

	interface OrderByCondition {
		column: string;
		direction: 'ASC' | 'DESC';
	}

	let filterModal = $state<{
		schema: string;
		table: string;
		columns: Column[];
		filters: FilterCondition[];
		filterLogic: 'AND' | 'OR';
		orderBy: OrderByCondition[];
		limit: number;
	} | null>(null);
	let isLoadingColumns = $state(false);

	const FILTER_OPERATORS = [
		{ value: '=', label: '= (equals)' },
		{ value: '!=', label: '!= (not equals)' },
		{ value: '>', label: '> (greater than)' },
		{ value: '>=', label: '>= (greater or equal)' },
		{ value: '<', label: '< (less than)' },
		{ value: '<=', label: '<= (less or equal)' },
		{ value: 'LIKE', label: 'LIKE (pattern match)' },
		{ value: 'ILIKE', label: 'ILIKE (case-insensitive)' },
		{ value: 'IS NULL', label: 'IS NULL' },
		{ value: 'IS NOT NULL', label: 'IS NOT NULL' },
		{ value: 'IN', label: 'IN (list)' },
	];

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

	// Expose toggleNode helper for E2E tests
	$effect(() => {
		if (typeof window !== 'undefined') {
			(window as any).__PGVOYAGER_E2E__ = (window as any).__PGVOYAGER_E2E__ || {};
			(window as any).__PGVOYAGER_E2E__.toggleNode = toggleNode;
		}
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

	function handleCopyName(node: SchemaTreeNode, quoted: boolean = false) {
		if (!node.schema) return;
		const fullName = quoted
			? `"${node.schema}"."${node.name}"`
			: `${node.schema}.${node.name}`;
		navigator.clipboard.writeText(fullName);
		closeContextMenu();
	}

	async function handleFilterTable(node: SchemaTreeNode) {
		if (!node.schema || !$activeConnectionId) return;
		closeContextMenu();

		isLoadingColumns = true;
		try {
			const columns = await loadTableColumns($activeConnectionId, node.schema, node.name);
			filterModal = {
				schema: node.schema,
				table: node.name,
				columns,
				filters: [{ column: columns.length > 0 ? columns[0].name : '', operator: '=', value: '' }],
				filterLogic: 'AND',
				orderBy: [],
				limit: 100
			};
		} catch (e) {
			console.error('Failed to load columns for filter:', e);
		} finally {
			isLoadingColumns = false;
		}
	}

	function closeFilterModal() {
		filterModal = null;
	}

	function addFilter() {
		if (!filterModal) return;
		const defaultColumn = filterModal.columns.length > 0 ? filterModal.columns[0].name : '';
		filterModal.filters = [...filterModal.filters, { column: defaultColumn, operator: '=', value: '' }];
	}

	function removeFilter(index: number) {
		if (!filterModal || filterModal.filters.length <= 1) return;
		filterModal.filters = filterModal.filters.filter((_, i) => i !== index);
	}

	function addOrderBy() {
		if (!filterModal) return;
		const defaultColumn = filterModal.columns.length > 0 ? filterModal.columns[0].name : '';
		filterModal.orderBy = [...filterModal.orderBy, { column: defaultColumn, direction: 'ASC' }];
	}

	function removeOrderBy(index: number) {
		if (!filterModal) return;
		filterModal.orderBy = filterModal.orderBy.filter((_, i) => i !== index);
	}

	function buildFilterCondition(filter: FilterCondition): string {
		const { column, operator, value } = filter;
		if (!column || !operator) return '';

		if (operator === 'IS NULL' || operator === 'IS NOT NULL') {
			return `"${column}" ${operator}`;
		} else if (operator === 'IN') {
			const values = value.split(',').map(v => `'${v.trim()}'`).join(', ');
			return `"${column}" IN (${values})`;
		} else if (operator === 'LIKE' || operator === 'ILIKE') {
			return `"${column}" ${operator} '${value}'`;
		} else {
			const isNumeric = !isNaN(Number(value)) && value.trim() !== '';
			const formattedValue = isNumeric ? value : `'${value}'`;
			return `"${column}" ${operator} ${formattedValue}`;
		}
	}

	function applyFilter() {
		if (!filterModal) return;

		const { schema, table, filters, filterLogic, orderBy, limit } = filterModal;

		// Build WHERE clause from multiple filters
		const validFilters = filters
			.map(f => buildFilterCondition(f))
			.filter(c => c !== '');

		let whereClause = '';
		if (validFilters.length > 0) {
			whereClause = `WHERE ${validFilters.join(`\n  ${filterLogic} `)}`;
		}

		// Build ORDER BY clause from multiple columns
		let orderClause = '';
		if (orderBy.length > 0) {
			const orderParts = orderBy.map(o => `"${o.column}" ${o.direction}`);
			orderClause = `ORDER BY ${orderParts.join(', ')}`;
		}

		const sql = `SELECT *
FROM "${schema}"."${table}"
${whereClause}
${orderClause}
LIMIT ${limit};`.replace(/\n\n+/g, '\n').trim();

		tabs.openQuery({ title: `Filter ${table}`, initialSql: sql });
		closeFilterModal();
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

	// ── Create Schema modal ──
	let createSchemaModal = $state<{ name: string } | null>(null);
	let isCreatingSchema = $state(false);
	let createSchemaError = $state<string | null>(null);

	function openCreateSchemaModal() {
		createSchemaModal = { name: '' };
		createSchemaError = null;
	}

	function closeCreateSchemaModal() {
		createSchemaModal = null;
		createSchemaError = null;
	}

	async function confirmCreateSchema() {
		if (!createSchemaModal || !$activeConnectionId) return;
		if (!createSchemaModal.name.trim()) {
			createSchemaError = 'Schema name is required';
			return;
		}

		isCreatingSchema = true;
		createSchemaError = null;

		try {
			await dataApi.createSchema($activeConnectionId, createSchemaModal.name.trim());
			closeCreateSchemaModal();
			refreshSchema();
		} catch (e) {
			createSchemaError = e instanceof Error ? e.message : 'Failed to create schema';
		} finally {
			isCreatingSchema = false;
		}
	}

	// ── Drop Schema modal ──
	let dropSchemaModal = $state<{ schema: string; cascade: boolean } | null>(null);
	let isDroppingSchema = $state(false);
	let dropSchemaError = $state<string | null>(null);

	function handleDropSchemaClick(node: SchemaTreeNode) {
		dropSchemaModal = { schema: node.name, cascade: false };
		dropSchemaError = null;
		closeContextMenu();
	}

	function closeDropSchemaModal() {
		dropSchemaModal = null;
		dropSchemaError = null;
	}

	async function confirmDropSchema() {
		if (!dropSchemaModal || !$activeConnectionId) return;

		isDroppingSchema = true;
		dropSchemaError = null;

		try {
			await dataApi.dropSchema($activeConnectionId, dropSchemaModal.schema, dropSchemaModal.cascade);
			closeDropSchemaModal();
			refreshSchema();
		} catch (e) {
			dropSchemaError = e instanceof Error ? e.message : 'Failed to drop schema';
		} finally {
			isDroppingSchema = false;
		}
	}

	// ── Create Table modal ──
	interface CreateTableColumn {
		name: string;
		type: string;
		nullable: boolean;
		defaultVal: string;
		primaryKey: boolean;
	}

	const PG_TYPES = [
		'INTEGER', 'BIGINT', 'SERIAL', 'TEXT', 'VARCHAR(255)', 'BOOLEAN',
		'NUMERIC(10,2)', 'DATE', 'TIMESTAMP', 'TIMESTAMPTZ', 'JSON', 'JSONB', 'XML', 'UUID'
	];

	let createTableModal = $state<{ schema: string; name: string; columns: CreateTableColumn[] } | null>(null);
	let isCreatingTable = $state(false);
	let createTableError = $state<string | null>(null);

	function handleCreateTableClick(node: SchemaTreeNode) {
		createTableModal = {
			schema: node.name,
			name: '',
			columns: [{ name: '', type: 'INTEGER', nullable: true, defaultVal: '', primaryKey: false }]
		};
		createTableError = null;
		closeContextMenu();
	}

	function closeCreateTableModal() {
		createTableModal = null;
		createTableError = null;
	}

	function addTableColumn() {
		if (!createTableModal) return;
		createTableModal.columns = [...createTableModal.columns, { name: '', type: 'INTEGER', nullable: true, defaultVal: '', primaryKey: false }];
	}

	function removeTableColumn(index: number) {
		if (!createTableModal || createTableModal.columns.length <= 1) return;
		createTableModal.columns = createTableModal.columns.filter((_, i) => i !== index);
	}

	async function confirmCreateTable() {
		if (!createTableModal || !$activeConnectionId) return;
		if (!createTableModal.name.trim()) {
			createTableError = 'Table name is required';
			return;
		}
		if (createTableModal.columns.some(c => !c.name.trim())) {
			createTableError = 'All column names are required';
			return;
		}

		isCreatingTable = true;
		createTableError = null;

		try {
			await dataApi.createTable($activeConnectionId, createTableModal.schema, {
				name: createTableModal.name.trim(),
				columns: createTableModal.columns.map(c => ({
					name: c.name.trim(),
					type: c.type,
					nullable: c.nullable,
					default: c.defaultVal || undefined,
					primaryKey: c.primaryKey
				}))
			});
			closeCreateTableModal();
			refreshSchema();
		} catch (e) {
			createTableError = e instanceof Error ? e.message : 'Failed to create table';
		} finally {
			isCreatingTable = false;
		}
	}

	// ── Add Constraint modal ──
	let addConstraintModal = $state<{
		schema: string;
		table: string;
		type: 'fk' | 'unique' | 'check';
		name: string;
		columns: string;
		refSchema: string;
		refTable: string;
		refColumns: string;
		onDelete: string;
		onUpdate: string;
		expression: string;
	} | null>(null);
	let isAddingConstraint = $state(false);
	let addConstraintError = $state<string | null>(null);

	function handleAddConstraintClick(node: SchemaTreeNode) {
		if (!node.schema) return;
		addConstraintModal = {
			schema: node.schema,
			table: node.name,
			type: 'unique',
			name: '',
			columns: '',
			refSchema: '',
			refTable: '',
			refColumns: '',
			onDelete: '',
			onUpdate: '',
			expression: ''
		};
		addConstraintError = null;
		closeContextMenu();
	}

	function closeAddConstraintModal() {
		addConstraintModal = null;
		addConstraintError = null;
	}

	async function confirmAddConstraint() {
		if (!addConstraintModal || !$activeConnectionId) return;

		isAddingConstraint = true;
		addConstraintError = null;

		try {
			const cols = addConstraintModal.columns.split(',').map(c => c.trim()).filter(Boolean);
			const refCols = addConstraintModal.refColumns.split(',').map(c => c.trim()).filter(Boolean);

			await dataApi.addConstraint(
				$activeConnectionId,
				addConstraintModal.schema,
				addConstraintModal.table,
				{
					type: addConstraintModal.type,
					name: addConstraintModal.name || undefined,
					columns: cols.length > 0 ? cols : undefined,
					refSchema: addConstraintModal.refSchema || undefined,
					refTable: addConstraintModal.refTable || undefined,
					refColumns: refCols.length > 0 ? refCols : undefined,
					onDelete: addConstraintModal.onDelete || undefined,
					onUpdate: addConstraintModal.onUpdate || undefined,
					expression: addConstraintModal.expression || undefined
				}
			);
			closeAddConstraintModal();
			refreshSchema();
		} catch (e) {
			addConstraintError = e instanceof Error ? e.message : 'Failed to add constraint';
		} finally {
			isAddingConstraint = false;
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
			<button class="context-menu-item" onclick={() => handleCopyName(menuNode, false)}>
				<Icon name="copy" size={14} />
				Copy name (schema.table)
			</button>
			<button class="context-menu-item" onclick={() => handleCopyName(menuNode, true)}>
				<Icon name="copy" size={14} />
				Copy name ("schema"."table")
			</button>
			<div class="context-menu-separator"></div>
			<button class="context-menu-item" onclick={() => handleAddConstraintClick(menuNode)}>
				<Icon name="lock" size={14} />
				Add Constraint...
			</button>
			<button class="context-menu-item context-menu-item-danger" onclick={() => handleDropTableClick(menuNode)}>
				<Icon name="trash" size={14} />
				Drop table...
			</button>
		{:else if contextMenu.menuType === 'schema'}
			<button class="context-menu-item" onclick={() => { openCreateSchemaModal(); closeContextMenu(); }}>
				<Icon name="plus" size={14} />
				Create Schema...
			</button>
			<button class="context-menu-item" onclick={() => handleCreateTableClick(menuNode)}>
				<Icon name="table" size={14} />
				Create Table...
			</button>
			<button class="context-menu-item" onclick={() => handleViewSchemaERD(menuNode)}>
				<Icon name="share-2" size={14} />
				View Schema ERD
			</button>
			<div class="context-menu-separator"></div>
			<button class="context-menu-item" onclick={() => { navigator.clipboard.writeText(menuNode.name); closeContextMenu(); }}>
				<Icon name="copy" size={14} />
				Copy schema name
			</button>
			<div class="context-menu-separator"></div>
			<button class="context-menu-item context-menu-item-danger" onclick={() => handleDropSchemaClick(menuNode)}>
				<Icon name="trash" size={14} />
				Drop Schema...
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

<!-- Filter Table Modal -->
{#if filterModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="modal-backdrop" onclick={closeFilterModal}></div>
	<div class="modal filter-modal">
		<div class="modal-header">
			<Icon name="filter" size={20} />
			<h3>Filter Table</h3>
		</div>
		<div class="modal-body">
			<p class="filter-table-name">"{filterModal.schema}"."{filterModal.table}"</p>

			<div class="filter-form">
				<!-- Filter conditions section -->
				<div class="filter-section">
					<div class="filter-section-header">
						<span class="filter-section-title">WHERE conditions</span>
						{#if filterModal.filters.length > 1}
							<select class="logic-select" bind:value={filterModal.filterLogic}>
								<option value="AND">AND</option>
								<option value="OR">OR</option>
							</select>
						{/if}
					</div>

					{#each filterModal.filters as filter, index}
						<div class="filter-condition">
							<select bind:value={filter.column} class="filter-col-select">
								{#each filterModal.columns as col}
									<option value={col.name}>{col.name}</option>
								{/each}
							</select>
							<select bind:value={filter.operator} class="filter-op-select">
								{#each FILTER_OPERATORS as op}
									<option value={op.value}>{op.value}</option>
								{/each}
							</select>
							{#if filter.operator !== 'IS NULL' && filter.operator !== 'IS NOT NULL'}
								<input
									type="text"
									bind:value={filter.value}
									class="filter-value-input"
									placeholder={filter.operator === 'LIKE' || filter.operator === 'ILIKE' ? '%pattern%' : filter.operator === 'IN' ? 'val1, val2, ...' : 'value'}
								/>
							{:else}
								<span class="filter-value-placeholder"></span>
							{/if}
							<button
								class="btn-icon"
								onclick={() => removeFilter(index)}
								disabled={filterModal.filters.length <= 1}
								title="Remove condition"
							>
								<Icon name="x" size={14} />
							</button>
						</div>
					{/each}

					<button class="btn btn-ghost btn-sm add-btn" onclick={addFilter}>
						<Icon name="plus" size={14} />
						Add condition
					</button>
				</div>

				<div class="filter-divider"></div>

				<!-- Order by section -->
				<div class="filter-section">
					<div class="filter-section-header">
						<span class="filter-section-title">ORDER BY</span>
					</div>

					{#each filterModal.orderBy as order, index}
						<div class="filter-condition">
							<select bind:value={order.column} class="filter-col-select flex-1">
								{#each filterModal.columns as col}
									<option value={col.name}>{col.name}</option>
								{/each}
							</select>
							<select bind:value={order.direction} class="filter-dir-select">
								<option value="ASC">ASC</option>
								<option value="DESC">DESC</option>
							</select>
							<button class="btn-icon" onclick={() => removeOrderBy(index)} title="Remove">
								<Icon name="x" size={14} />
							</button>
						</div>
					{/each}

					{#if filterModal.orderBy.length === 0}
						<p class="filter-empty-text">No ordering (default)</p>
					{/if}

					<button class="btn btn-ghost btn-sm add-btn" onclick={addOrderBy}>
						<Icon name="plus" size={14} />
						Add order column
					</button>
				</div>

				<div class="filter-divider"></div>

				<div class="filter-row">
					<label>
						<span>Limit</span>
						<input type="number" bind:value={filterModal.limit} min="1" max="10000" />
					</label>
				</div>
			</div>
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary btn-sm" onclick={closeFilterModal}>
				Cancel
			</button>
			<button class="btn btn-primary btn-sm" onclick={applyFilter}>
				<Icon name="play" size={14} />
				Open in Query Editor
			</button>
		</div>
	</div>
{/if}

<!-- Create Schema Modal -->
{#if createSchemaModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="modal-backdrop" onclick={closeCreateSchemaModal}></div>
	<div class="modal" data-testid="create-schema-modal">
		<div class="modal-header modal-header-primary">
			<Icon name="folder" size={20} />
			<h3>Create Schema</h3>
		</div>
		<div class="modal-body">
			<label class="form-field">
				<span>Schema Name</span>
				<input
					type="text"
					bind:value={createSchemaModal.name}
					placeholder="new_schema"
					data-testid="input-schema-name"
					onkeydown={(e) => { if (e.key === 'Enter') confirmCreateSchema(); }}
				/>
			</label>

			{#if createSchemaError}
				<div class="modal-error">
					<Icon name="alert-circle" size={14} />
					{createSchemaError}
				</div>
			{/if}
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary btn-sm" onclick={closeCreateSchemaModal} disabled={isCreatingSchema}>
				Cancel
			</button>
			<button class="btn btn-primary btn-sm" onclick={confirmCreateSchema} disabled={isCreatingSchema} data-testid="btn-confirm-create-schema">
				{#if isCreatingSchema}
					<Icon name="refresh" size={14} class="spinning" />
					Creating...
				{:else}
					Create Schema
				{/if}
			</button>
		</div>
	</div>
{/if}

<!-- Drop Schema Modal -->
{#if dropSchemaModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="modal-backdrop" onclick={closeDropSchemaModal}></div>
	<div class="modal" data-testid="drop-schema-modal">
		<div class="modal-header">
			<Icon name="alert-circle" size={20} />
			<h3>Drop Schema</h3>
		</div>
		<div class="modal-body">
			<p>Are you sure you want to drop the schema <strong>"{dropSchemaModal.schema}"</strong>?</p>
			<p class="warning-text">This action cannot be undone!</p>

			{#if dropSchemaError}
				<div class="modal-error">
					<Icon name="alert-circle" size={14} />
					{dropSchemaError}
				</div>
			{/if}

			<label class="cascade-option">
				<input type="checkbox" bind:checked={dropSchemaModal.cascade} data-testid="checkbox-cascade" />
				<span>CASCADE (also drop all contained objects)</span>
			</label>
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary btn-sm" onclick={closeDropSchemaModal} disabled={isDroppingSchema}>
				Cancel
			</button>
			<button class="btn btn-danger btn-sm" onclick={confirmDropSchema} disabled={isDroppingSchema} data-testid="btn-confirm-drop-schema">
				{#if isDroppingSchema}
					<Icon name="refresh" size={14} class="spinning" />
					Dropping...
				{:else}
					Drop Schema
				{/if}
			</button>
		</div>
	</div>
{/if}

<!-- Create Table Modal -->
{#if createTableModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="modal-backdrop" onclick={closeCreateTableModal}></div>
	<div class="modal create-table-modal" data-testid="create-table-modal">
		<div class="modal-header modal-header-primary">
			<Icon name="table" size={20} />
			<h3>Create Table</h3>
		</div>
		<div class="modal-body">
			<p class="filter-table-name">Schema: {createTableModal.schema}</p>

			<label class="form-field">
				<span>Table Name</span>
				<input
					type="text"
					bind:value={createTableModal.name}
					placeholder="new_table"
					data-testid="input-table-name"
				/>
			</label>

			<div class="filter-section" style="margin-top: 12px">
				<div class="filter-section-header">
					<span class="filter-section-title">Columns</span>
				</div>

				{#each createTableModal.columns as col, index}
					<div class="table-col-row">
						<input
							type="text"
							bind:value={col.name}
							placeholder="column_name"
							class="col-name-input"
							data-testid="input-col-name-{index}"
						/>
						<select bind:value={col.type} class="col-type-select" data-testid="select-col-type-{index}">
							{#each PG_TYPES as pgType}
								<option value={pgType}>{pgType}</option>
							{/each}
						</select>
						<label class="col-option" title="Nullable">
							<input type="checkbox" bind:checked={col.nullable} />
							<span>NULL</span>
						</label>
						<label class="col-option" title="Primary Key">
							<input type="checkbox" bind:checked={col.primaryKey} data-testid="checkbox-pk-{index}" />
							<span>PK</span>
						</label>
						<button
							class="btn-icon"
							onclick={() => removeTableColumn(index)}
							disabled={createTableModal.columns.length <= 1}
							title="Remove column"
						>
							<Icon name="x" size={14} />
						</button>
					</div>
				{/each}

				<button class="btn btn-ghost btn-sm add-btn" onclick={addTableColumn} data-testid="btn-add-column">
					<Icon name="plus" size={14} />
					Add column
				</button>
			</div>

			{#if createTableError}
				<div class="modal-error">
					<Icon name="alert-circle" size={14} />
					{createTableError}
				</div>
			{/if}
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary btn-sm" onclick={closeCreateTableModal} disabled={isCreatingTable}>
				Cancel
			</button>
			<button class="btn btn-primary btn-sm" onclick={confirmCreateTable} disabled={isCreatingTable} data-testid="btn-confirm-create-table">
				{#if isCreatingTable}
					<Icon name="refresh" size={14} class="spinning" />
					Creating...
				{:else}
					Create Table
				{/if}
			</button>
		</div>
	</div>
{/if}

<!-- Add Constraint Modal -->
{#if addConstraintModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="modal-backdrop" onclick={closeAddConstraintModal}></div>
	<div class="modal constraint-modal" data-testid="add-constraint-modal">
		<div class="modal-header modal-header-primary">
			<Icon name="lock" size={20} />
			<h3>Add Constraint</h3>
		</div>
		<div class="modal-body">
			<p class="filter-table-name">"{addConstraintModal.schema}"."{addConstraintModal.table}"</p>

			<div class="form-grid">
				<label class="form-field">
					<span>Constraint Type</span>
					<select bind:value={addConstraintModal.type} data-testid="select-constraint-type">
						<option value="unique">UNIQUE</option>
						<option value="fk">FOREIGN KEY</option>
						<option value="check">CHECK</option>
					</select>
				</label>

				<label class="form-field">
					<span>Constraint Name (optional)</span>
					<input type="text" bind:value={addConstraintModal.name} placeholder="auto-generated" data-testid="input-constraint-name" />
				</label>

				{#if addConstraintModal.type !== 'check'}
					<label class="form-field">
						<span>Columns (comma-separated)</span>
						<input type="text" bind:value={addConstraintModal.columns} placeholder="col1, col2" data-testid="input-constraint-columns" />
					</label>
				{/if}

				{#if addConstraintModal.type === 'fk'}
					<label class="form-field">
						<span>Reference Schema (optional)</span>
						<input type="text" bind:value={addConstraintModal.refSchema} placeholder="public" />
					</label>
					<label class="form-field">
						<span>Reference Table</span>
						<input type="text" bind:value={addConstraintModal.refTable} placeholder="referenced_table" data-testid="input-ref-table" />
					</label>
					<label class="form-field">
						<span>Reference Columns</span>
						<input type="text" bind:value={addConstraintModal.refColumns} placeholder="id" data-testid="input-ref-columns" />
					</label>
					<label class="form-field">
						<span>ON DELETE</span>
						<select bind:value={addConstraintModal.onDelete}>
							<option value="">None</option>
							<option value="CASCADE">CASCADE</option>
							<option value="SET NULL">SET NULL</option>
							<option value="SET DEFAULT">SET DEFAULT</option>
							<option value="RESTRICT">RESTRICT</option>
							<option value="NO ACTION">NO ACTION</option>
						</select>
					</label>
					<label class="form-field">
						<span>ON UPDATE</span>
						<select bind:value={addConstraintModal.onUpdate}>
							<option value="">None</option>
							<option value="CASCADE">CASCADE</option>
							<option value="SET NULL">SET NULL</option>
							<option value="SET DEFAULT">SET DEFAULT</option>
							<option value="RESTRICT">RESTRICT</option>
							<option value="NO ACTION">NO ACTION</option>
						</select>
					</label>
				{/if}

				{#if addConstraintModal.type === 'check'}
					<label class="form-field">
						<span>CHECK Expression</span>
						<input type="text" bind:value={addConstraintModal.expression} placeholder="price > 0" data-testid="input-check-expression" />
					</label>
				{/if}
			</div>

			{#if addConstraintError}
				<div class="modal-error">
					<Icon name="alert-circle" size={14} />
					{addConstraintError}
				</div>
			{/if}
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary btn-sm" onclick={closeAddConstraintModal} disabled={isAddingConstraint}>
				Cancel
			</button>
			<button class="btn btn-primary btn-sm" onclick={confirmAddConstraint} disabled={isAddingConstraint} data-testid="btn-confirm-add-constraint">
				{#if isAddingConstraint}
					<Icon name="refresh" size={14} class="spinning" />
					Adding...
				{:else}
					Add Constraint
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

	/* Filter Modal */
	.filter-modal {
		width: 520px;
		max-height: 80vh;
	}

	.filter-modal .modal-body {
		overflow-y: auto;
		max-height: 60vh;
	}

	.filter-table-name {
		font-family: var(--font-mono);
		font-size: 14px;
		font-weight: 600;
		color: var(--color-primary);
		margin-bottom: 16px !important;
	}

	.filter-form {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.filter-section {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.filter-section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.filter-section-title {
		font-size: 12px;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.logic-select {
		padding: 2px 8px;
		font-size: 11px;
		font-weight: 600;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-surface);
		color: var(--color-primary);
	}

	.filter-condition {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.filter-col-select {
		flex: 1;
		min-width: 120px;
	}

	.filter-op-select {
		width: 90px;
	}

	.filter-value-input {
		flex: 1;
		min-width: 100px;
	}

	.filter-value-placeholder {
		flex: 1;
		min-width: 100px;
	}

	.filter-dir-select {
		width: 70px;
	}

	.filter-condition select,
	.filter-condition input {
		padding: 6px 8px;
		font-size: 12px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.filter-condition select:focus,
	.filter-condition input:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.filter-condition input::placeholder {
		color: var(--color-text-dim);
	}

	.btn-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}

	.btn-icon:hover:not(:disabled) {
		background: var(--color-surface);
		color: var(--color-error);
	}

	.btn-icon:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	.add-btn {
		align-self: flex-start;
		margin-top: 4px;
	}

	.filter-empty-text {
		font-size: 12px;
		color: var(--color-text-dim);
		font-style: italic;
		margin: 0;
	}

	.filter-row label {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.filter-row label span {
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text-muted);
	}

	.filter-row select,
	.filter-row input {
		padding: 8px 10px;
		font-size: 13px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.filter-row select:focus,
	.filter-row input:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.filter-row input::placeholder {
		color: var(--color-text-dim);
	}

	.filter-divider {
		height: 1px;
		background: var(--color-border);
		margin: 8px 0;
	}

	.flex-1 {
		flex: 1;
	}

	/* Primary modal header (non-destructive actions) */
	.modal-header-primary {
		color: var(--color-primary);
	}

	/* Form fields */
	.form-field {
		display: flex;
		flex-direction: column;
		gap: 4px;
		margin-bottom: 12px;
	}

	.form-field span {
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text-muted);
	}

	.form-field input,
	.form-field select {
		padding: 8px 10px;
		font-size: 13px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.form-field input:focus,
	.form-field select:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.form-field input::placeholder {
		color: var(--color-text-dim);
	}

	.form-grid {
		display: flex;
		flex-direction: column;
	}

	/* Create Table Modal */
	.create-table-modal {
		width: 600px;
		max-height: 80vh;
	}

	.create-table-modal .modal-body {
		overflow-y: auto;
		max-height: 60vh;
	}

	.table-col-row {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-bottom: 6px;
	}

	.col-name-input {
		flex: 1;
		min-width: 120px;
		padding: 6px 8px;
		font-size: 12px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.col-name-input:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.col-name-input::placeholder {
		color: var(--color-text-dim);
	}

	.col-type-select {
		width: 140px;
		padding: 6px 8px;
		font-size: 12px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.col-type-select:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.col-option {
		display: flex;
		align-items: center;
		gap: 3px;
		font-size: 11px;
		color: var(--color-text-muted);
		cursor: pointer;
		white-space: nowrap;
	}

	.col-option input {
		width: 14px;
		height: 14px;
		accent-color: var(--color-primary);
	}

	/* Add Constraint Modal */
	.constraint-modal {
		width: 480px;
		max-height: 80vh;
	}

	.constraint-modal .modal-body {
		overflow-y: auto;
		max-height: 60vh;
	}
</style>
