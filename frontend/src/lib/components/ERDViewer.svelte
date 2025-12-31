<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { activeConnectionId } from '$lib/stores/connections';
	import { tabs } from '$lib/stores/tabs';
	import { schemaApi } from '$lib/api/client';
	import type { Tab, SchemaRelationship, Table, ERDLocation } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';
	import cytoscape from 'cytoscape';
	import svg from 'cytoscape-svg';
	import type { Core, ElementDefinition, LayoutOptions } from 'cytoscape';

	// Register cytoscape-svg extension
	cytoscape.use(svg);

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let container: HTMLDivElement | undefined = $state();
	let cy: Core | null = $state(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let relationships = $state<SchemaRelationship[]>([]);
	let tables = $state<Table[]>([]);
	let showExportMenu = $state(false);
	let isRendering = false; // Guard against re-render loops
	let lastRenderedLocation: string | null = null;

	// Navigation state - computed from tabs store
	let canGoBack = $derived(tabs.canNavigateERDBack(tab.id));
	let canGoForward = $derived(tabs.canNavigateERDForward(tab.id));
	let currentLocation = $derived(tabs.getCurrentERDLocation(tab.id));
	let isFullSchemaView = $derived(!currentLocation?.centeredTable);

	// Load data when tab.schema or connection changes
	$effect(() => {
		const schema = tab.schema;
		const connId = $activeConnectionId;
		if (schema && connId) {
			loadERDData(connId, schema);
		}
	});

	// Re-render graph when location changes (but not during initial load)
	$effect(() => {
		const location = currentLocation;
		const locationKey = location ? `${location.schema}:${location.centeredTable || 'full'}` : null;

		// Only re-render if location actually changed and we have data
		if (location && relationships.length > 0 && !isLoading && locationKey !== lastRenderedLocation) {
			renderGraph();
		}
	});

	async function loadERDData(connId: string, schema: string) {
		isLoading = true;
		error = null;

		try {
			const [relData, tableData] = await Promise.all([
				schemaApi.getSchemaRelationships(connId, schema),
				schemaApi.listTables(connId, schema)
			]);

			relationships = relData || [];
			tables = tableData || [];

			// Wait for container to be available, then render
			await new Promise(resolve => setTimeout(resolve, 50));
			renderGraph();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load ERD data';
		} finally {
			isLoading = false;
		}
	}

	function renderGraph() {
		if (!container) return;
		if (isRendering) return; // Prevent concurrent renders

		const location = currentLocation;
		if (!location) return;

		isRendering = true;
		const locationKey = `${location.schema}:${location.centeredTable || 'full'}`;

		let relevantTables: Set<string>;
		let relevantRelationships: SchemaRelationship[];

		if (location.centeredTable) {
			// Table-centered view: show only directly related tables (1-hop)
			const centerTable = location.centeredTable;
			relevantTables = new Set([centerTable]);
			relevantRelationships = relationships.filter(r =>
				(r.sourceSchema === tab.schema && r.sourceTable === centerTable) ||
				(r.targetSchema === tab.schema && r.targetTable === centerTable)
			);

			// Add related tables
			for (const r of relevantRelationships) {
				if (r.sourceSchema === tab.schema) relevantTables.add(r.sourceTable);
				if (r.targetSchema === tab.schema) relevantTables.add(r.targetTable);
			}
		} else {
			// Full schema view: show all tables
			relevantTables = new Set(tables.map(t => t.name));
			relevantRelationships = relationships.filter(r =>
				r.sourceSchema === tab.schema && r.targetSchema === tab.schema
			);
		}

		// Build Cytoscape elements
		const elements: ElementDefinition[] = [];

		// Add nodes (tables)
		for (const tableName of relevantTables) {
			const tableInfo = tables.find(t => t.name === tableName);
			elements.push({
				data: {
					id: tableName,
					label: tableName,
					hasPk: tableInfo?.hasPk ?? false,
					rowCount: tableInfo?.rowCount ?? 0,
					isCentered: tableName === location.centeredTable
				}
			});
		}

		// Add edges (relationships)
		for (const rel of relevantRelationships) {
			if (relevantTables.has(rel.sourceTable) && relevantTables.has(rel.targetTable)) {
				elements.push({
					data: {
						id: `${rel.constraintName}-${rel.sourceTable}-${rel.targetTable}`,
						source: rel.sourceTable,
						target: rel.targetTable,
						label: rel.sourceColumns.join(', '),
						constraintName: rel.constraintName
					}
				});
			}
		}

		// Destroy existing instance
		if (cy) {
			cy.destroy();
		}

		// Initialize Cytoscape
		cy = cytoscape({
			container,
			elements,
			style: getGraphStyle(),
			layout: getLayout(location.centeredTable),
			minZoom: 0.1,
			maxZoom: 3,
			wheelSensitivity: 0.3
		});

		// Event handlers
		cy.on('tap', 'node', (e) => {
			const nodeId = e.target.id();
			if (nodeId !== location.centeredTable) {
				// Single click: recenter ERD on this table
				tabs.navigateERD(tab.id, tab.schema!, nodeId);
			}
		});

		cy.on('dbltap', 'node', (e) => {
			const nodeId = e.target.id();
			// Double click: open table in TableViewer
			tabs.openTable(tab.schema!, nodeId);
		});

		// Mark rendering complete
		lastRenderedLocation = locationKey;
		isRendering = false;
	}

	function getGraphStyle() {
		return [
			{
				selector: 'node',
				style: {
					'label': 'data(label)',
					'text-valign': 'center',
					'text-halign': 'center',
					'background-color': '#313244',
					'border-color': '#45475a',
					'border-width': 2,
					'color': '#cdd6f4',
					'font-size': '12px',
					'font-family': 'ui-monospace, monospace',
					'width': 'label',
					'height': 30,
					'padding': '12px',
					'shape': 'roundrectangle',
					'text-wrap': 'none'
				}
			},
			{
				selector: 'node[?isCentered]',
				style: {
					'background-color': '#89b4fa',
					'color': '#1e1e2e',
					'border-color': '#74c7ec',
					'border-width': 3,
					'font-weight': 'bold'
				}
			},
			{
				selector: 'edge',
				style: {
					'width': 2,
					'line-color': '#6c7086',
					'target-arrow-color': '#6c7086',
					'target-arrow-shape': 'triangle',
					'arrow-scale': 1,
					'curve-style': 'bezier',
					'label': 'data(label)',
					'font-size': '10px',
					'font-family': 'ui-monospace, monospace',
					'color': '#a6adc8',
					'text-rotation': 'autorotate',
					'text-margin-y': -10,
					'text-background-color': '#1e1e2e',
					'text-background-opacity': 0.8,
					'text-background-padding': '2px'
				}
			},
			{
				selector: 'edge:selected',
				style: {
					'line-color': '#89b4fa',
					'target-arrow-color': '#89b4fa',
					'width': 3
				}
			},
			{
				selector: 'node:selected',
				style: {
					'border-color': '#f9e2af',
					'border-width': 3
				}
			}
		];
	}

	function getLayout(centeredTable?: string): LayoutOptions {
		if (centeredTable) {
			// Concentric layout for table-centered view
			return {
				name: 'concentric',
				concentric: (node: { id: () => string }) => node.id() === centeredTable ? 2 : 1,
				levelWidth: () => 1,
				minNodeSpacing: 80,
				animate: true,
				animationDuration: 300
			} as LayoutOptions;
		} else {
			// Grid layout for full schema (fast, non-blocking)
			return {
				name: 'grid',
				rows: Math.ceil(Math.sqrt(tables.length || 10)),
				animate: false
			} as LayoutOptions;
		}
	}

	function handleBack() {
		tabs.navigateERDBack(tab.id);
	}

	function handleForward() {
		tabs.navigateERDForward(tab.id);
	}

	function handleViewFullSchema() {
		tabs.navigateERD(tab.id, tab.schema!);
	}

	function handleFitToScreen() {
		cy?.fit(undefined, 50);
	}

	function handleZoomIn() {
		if (cy) {
			cy.zoom(cy.zoom() * 1.3);
			cy.center();
		}
	}

	function handleZoomOut() {
		if (cy) {
			cy.zoom(cy.zoom() / 1.3);
			cy.center();
		}
	}

	function handleRefresh() {
		loadERDData();
	}

	// Export functions
	function downloadFile(content: string | Blob, filename: string, mimeType: string) {
		const blob = content instanceof Blob ? content : new Blob([content], { type: mimeType });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = filename;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
		showExportMenu = false;
	}

	function exportPNG() {
		if (!cy) return;
		const png = cy.png({ full: true, scale: 2, bg: '#1e1e2e' });
		// Convert base64 to blob
		const byteString = atob(png.split(',')[1]);
		const mimeString = png.split(',')[0].split(':')[1].split(';')[0];
		const ab = new ArrayBuffer(byteString.length);
		const ia = new Uint8Array(ab);
		for (let i = 0; i < byteString.length; i++) {
			ia[i] = byteString.charCodeAt(i);
		}
		const blob = new Blob([ab], { type: mimeString });
		const filename = currentLocation?.centeredTable
			? `erd-${tab.schema}-${currentLocation.centeredTable}.png`
			: `erd-${tab.schema}.png`;
		downloadFile(blob, filename, 'image/png');
	}

	function exportSVG() {
		if (!cy) return;
		const svg = cy.svg({ full: true, scale: 1, bg: '#1e1e2e' });
		const filename = currentLocation?.centeredTable
			? `erd-${tab.schema}-${currentLocation.centeredTable}.svg`
			: `erd-${tab.schema}.svg`;
		downloadFile(svg, filename, 'image/svg+xml');
	}

	function exportJSON() {
		const location = currentLocation;
		if (!location) return;

		let relevantTables: string[];
		let relevantRelationships: SchemaRelationship[];

		if (location.centeredTable) {
			const centerTable = location.centeredTable;
			const tableSet = new Set([centerTable]);
			relevantRelationships = relationships.filter(r =>
				(r.sourceSchema === tab.schema && r.sourceTable === centerTable) ||
				(r.targetSchema === tab.schema && r.targetTable === centerTable)
			);
			for (const r of relevantRelationships) {
				if (r.sourceSchema === tab.schema) tableSet.add(r.sourceTable);
				if (r.targetSchema === tab.schema) tableSet.add(r.targetTable);
			}
			relevantTables = Array.from(tableSet);
		} else {
			relevantTables = tables.map(t => t.name);
			relevantRelationships = relationships.filter(r =>
				r.sourceSchema === tab.schema && r.targetSchema === tab.schema
			);
		}

		const data = {
			schema: tab.schema,
			centeredTable: location.centeredTable || null,
			tables: relevantTables,
			relationships: relevantRelationships
		};

		const filename = location.centeredTable
			? `erd-${tab.schema}-${location.centeredTable}.json`
			: `erd-${tab.schema}.json`;
		downloadFile(JSON.stringify(data, null, 2), filename, 'application/json');
	}

	function exportSQL() {
		const location = currentLocation;
		if (!location) return;

		let relevantRelationships: SchemaRelationship[];

		if (location.centeredTable) {
			const centerTable = location.centeredTable;
			relevantRelationships = relationships.filter(r =>
				(r.sourceSchema === tab.schema && r.sourceTable === centerTable) ||
				(r.targetSchema === tab.schema && r.targetTable === centerTable)
			);
		} else {
			relevantRelationships = relationships.filter(r =>
				r.sourceSchema === tab.schema && r.targetSchema === tab.schema
			);
		}

		let sql = `-- ERD Foreign Key Constraints for schema: ${tab.schema}\n`;
		sql += `-- Generated by PgVoyager\n\n`;

		for (const rel of relevantRelationships) {
			sql += `ALTER TABLE ${rel.sourceSchema}.${rel.sourceTable}\n`;
			sql += `  ADD CONSTRAINT ${rel.constraintName}\n`;
			sql += `  FOREIGN KEY (${rel.sourceColumns.join(', ')})\n`;
			sql += `  REFERENCES ${rel.targetSchema}.${rel.targetTable}(${rel.targetColumns.join(', ')})\n`;
			sql += `  ON UPDATE ${rel.onUpdate}\n`;
			sql += `  ON DELETE ${rel.onDelete};\n\n`;
		}

		const filename = location.centeredTable
			? `erd-${tab.schema}-${location.centeredTable}.sql`
			: `erd-${tab.schema}.sql`;
		downloadFile(sql, filename, 'text/sql');
	}

	// Close export menu on click outside
	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.export-dropdown')) {
			showExportMenu = false;
		}
	}

	onMount(() => {
		document.addEventListener('click', handleClickOutside);
	});

	onDestroy(() => {
		if (cy) {
			cy.destroy();
			cy = null;
		}
		document.removeEventListener('click', handleClickOutside);
	});
</script>

<div class="erd-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<div class="nav-buttons">
				<button class="btn btn-sm btn-ghost" onclick={handleBack} disabled={!canGoBack} title="Go Back">
					<Icon name="arrow-left" size={16} />
				</button>
				<button class="btn btn-sm btn-ghost" onclick={handleForward} disabled={!canGoForward} title="Go Forward">
					<Icon name="arrow-right" size={16} />
				</button>
			</div>
			<div class="breadcrumb">
				<Icon name="share-2" size={14} />
				<span class="erd-title">
					{isFullSchemaView ? `Schema: ${tab.schema}` : `${tab.schema}.${currentLocation?.centeredTable}`}
				</span>
			</div>
			{#if !isFullSchemaView}
				<button class="btn btn-sm btn-ghost" onclick={handleViewFullSchema}>
					View Full Schema
				</button>
			{/if}
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={handleZoomOut} title="Zoom Out">
				<Icon name="zoom-out" size={14} />
			</button>
			<button class="btn btn-sm btn-ghost" onclick={handleZoomIn} title="Zoom In">
				<Icon name="zoom-in" size={14} />
			</button>
			<button class="btn btn-sm btn-ghost" onclick={handleFitToScreen} title="Fit to Screen">
				<Icon name="maximize" size={14} />
			</button>
			<div class="export-dropdown">
				<button
					class="btn btn-sm btn-ghost"
					onclick={(e) => { e.stopPropagation(); showExportMenu = !showExportMenu; }}
					title="Export"
				>
					<Icon name="download" size={14} />
					<span>Export</span>
					<Icon name="chevron-down" size={12} />
				</button>
				{#if showExportMenu}
					<div class="export-menu">
						<button class="export-item" onclick={exportPNG}>
							<Icon name="image" size={14} />
							<span>PNG Image</span>
						</button>
						<button class="export-item" onclick={exportSVG}>
							<Icon name="file-code" size={14} />
							<span>SVG Vector</span>
						</button>
						<button class="export-item" onclick={exportJSON}>
							<Icon name="file-code" size={14} />
							<span>JSON Data</span>
						</button>
						<button class="export-item" onclick={exportSQL}>
							<Icon name="database" size={14} />
							<span>SQL DDL</span>
						</button>
					</div>
				{/if}
			</div>
			<button class="btn btn-sm btn-ghost" onclick={handleRefresh} disabled={isLoading} title="Refresh">
				<Icon name="refresh" size={14} class={isLoading ? 'spinning' : ''} />
			</button>
		</div>
	</div>

	{#if isLoading && !cy}
		<div class="loading">
			<Icon name="refresh" size={24} class="spinning" />
			<span>Loading ERD...</span>
		</div>
	{:else if error}
		<div class="error">
			<Icon name="alert-circle" size={24} />
			<span>{error}</span>
		</div>
	{:else if relationships.length === 0 && tables.length === 0}
		<div class="empty">
			<Icon name="share-2" size={48} />
			<span>No relationships found in this schema</span>
		</div>
	{:else}
		<div class="graph-container" bind:this={container}></div>
		<div class="graph-legend">
			<div class="legend-item">
				<span class="legend-dot centered"></span>
				<span>Centered table</span>
			</div>
			<div class="legend-item">
				<span class="legend-dot"></span>
				<span>Related table</span>
			</div>
			<div class="legend-hint">Click table to center | Double-click to open</div>
		</div>
	{/if}
</div>

<style>
	.erd-viewer {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
		background: var(--color-bg);
	}

	.toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 16px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
	}

	.toolbar-left,
	.toolbar-right {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.nav-buttons {
		display: flex;
		gap: 2px;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 0 8px;
	}

	.erd-title {
		font-weight: 600;
		font-family: var(--font-mono);
	}

	.graph-container {
		flex: 1;
		background: var(--color-bg);
	}

	.graph-legend {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 8px 16px;
		background: var(--color-bg-secondary);
		border-top: 1px solid var(--color-border);
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.legend-item {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.legend-dot {
		width: 12px;
		height: 12px;
		border-radius: 3px;
		background: #313244;
		border: 2px solid #45475a;
	}

	.legend-dot.centered {
		background: #89b4fa;
		border-color: #74c7ec;
	}

	.legend-hint {
		margin-left: auto;
		font-style: italic;
	}

	.loading,
	.error,
	.empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		color: var(--color-text-muted);
	}

	.error {
		color: var(--color-error);
	}

	.empty {
		opacity: 0.6;
	}

	.export-dropdown {
		position: relative;
	}

	.export-dropdown .btn {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.export-menu {
		position: absolute;
		top: 100%;
		right: 0;
		margin-top: 4px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		min-width: 150px;
		z-index: 100;
		overflow: hidden;
	}

	.export-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 8px 12px;
		background: none;
		border: none;
		color: var(--color-text);
		font-size: 13px;
		cursor: pointer;
		text-align: left;
	}

	.export-item:hover {
		background: var(--color-bg-hover);
	}

	:global(.spinning) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}
</style>
