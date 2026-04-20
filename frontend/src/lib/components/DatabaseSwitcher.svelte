<script lang="ts">
	import { activeConnection, activeConnectionId, connections } from '$lib/stores/connections';
	import { clearSchema, refreshSchema } from '$lib/stores/schema';
	import { connectionApi, schemaApi } from '$lib/api/client';
	import type { Database } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	let databases = $state<Database[]>([]);
	let isLoading = $state(false);
	let isExpanded = $state(true);
	let error = $state<string | null>(null);
	let loadedForConnId = $state<string | null>(null);
	let busyDatabase = $state<string | null>(null);

	let contextMenu = $state<{ database: string; x: number; y: number } | null>(null);

	let createModal = $state<{ name: string; owner: string; template: string } | null>(null);
	let createError = $state<string | null>(null);
	let isCreating = $state(false);

	let dropModal = $state<{ database: string; force: boolean } | null>(null);
	let dropError = $state<string | null>(null);
	let isDropping = $state(false);

	$effect(() => {
		const connId = $activeConnectionId;
		if (!connId) {
			databases = [];
			loadedForConnId = null;
			return;
		}
		if (connId !== loadedForConnId) {
			loadedForConnId = connId;
			loadDatabases();
		}
	});

	async function loadDatabases() {
		if (!$activeConnectionId) return;
		isLoading = true;
		error = null;
		try {
			const list = await schemaApi.listDatabases($activeConnectionId);
			databases = list || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load databases';
		} finally {
			isLoading = false;
		}
	}

	async function handleSwitch(dbName: string) {
		if (!$activeConnectionId || !$activeConnection) return;
		if (dbName === $activeConnection.database || busyDatabase) return;

		busyDatabase = dbName;
		error = null;
		try {
			const updated = await connectionApi.switchDatabase($activeConnectionId, dbName);
			connections.updateConnection(updated.id, updated);
			clearSchema();
			refreshSchema();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to switch database';
		} finally {
			busyDatabase = null;
		}
	}

	function openContextMenu(e: MouseEvent, dbName: string) {
		e.preventDefault();
		e.stopPropagation();
		contextMenu = { database: dbName, x: e.clientX, y: e.clientY };
	}

	function openKebab(e: MouseEvent, dbName: string) {
		e.stopPropagation();
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		contextMenu = { database: dbName, x: rect.right, y: rect.top };
	}

	function closeContextMenu() {
		contextMenu = null;
	}

	function openCreateModal() {
		createModal = { name: '', owner: '', template: '' };
		createError = null;
	}

	function closeCreateModal() {
		createModal = null;
		createError = null;
	}

	async function confirmCreate() {
		if (!createModal || !$activeConnectionId) return;
		const name = createModal.name.trim();
		if (!name) {
			createError = 'Database name is required';
			return;
		}
		isCreating = true;
		createError = null;
		try {
			await connectionApi.createDatabase($activeConnectionId, {
				name,
				owner: createModal.owner.trim() || undefined,
				template: createModal.template.trim() || undefined
			});
			closeCreateModal();
			await loadDatabases();
		} catch (e) {
			createError = e instanceof Error ? e.message : 'Failed to create database';
		} finally {
			isCreating = false;
		}
	}

	function openDropModal(dbName: string) {
		dropModal = { database: dbName, force: false };
		dropError = null;
		closeContextMenu();
	}

	function closeDropModal() {
		dropModal = null;
		dropError = null;
	}

	async function confirmDrop() {
		if (!dropModal || !$activeConnectionId) return;
		isDropping = true;
		dropError = null;
		try {
			const result = await connectionApi.dropDatabase(
				$activeConnectionId,
				dropModal.database,
				dropModal.force
			);
			// If we dropped the current DB, backend switched us to postgres.
			if ($activeConnection && $activeConnection.database !== result.currentDatabase) {
				connections.updateConnection($activeConnectionId, { database: result.currentDatabase });
				clearSchema();
				refreshSchema();
			}
			closeDropModal();
			await loadDatabases();
		} catch (e) {
			dropError = e instanceof Error ? e.message : 'Failed to drop database';
		} finally {
			isDropping = false;
		}
	}

	function handleCopyName(dbName: string) {
		navigator.clipboard.writeText(dbName);
		closeContextMenu();
	}

	$effect(() => {
		if (contextMenu) {
			const handler = () => closeContextMenu();
			const keyHandler = (e: KeyboardEvent) => {
				if (e.key === 'Escape') closeContextMenu();
			};
			document.addEventListener('click', handler);
			document.addEventListener('keydown', keyHandler);
			return () => {
				document.removeEventListener('click', handler);
				document.removeEventListener('keydown', keyHandler);
			};
		}
	});
</script>

{#if $activeConnection}
	<div class="databases-panel" data-testid="databases-panel">
		<div class="panel-header">
			<button
				class="panel-title"
				onclick={() => (isExpanded = !isExpanded)}
				data-testid="databases-panel-toggle"
				title={isExpanded ? 'Collapse' : 'Expand'}
			>
				<span class="panel-chevron" class:expanded={isExpanded}>
					<Icon name="chevron-right" size={10} />
				</span>
				<Icon name="database" size={12} />
				<span>Databases</span>
				{#if databases.length > 0}
					<span class="panel-count">{databases.length}</span>
				{/if}
			</button>
			<div class="panel-actions">
				<button
					class="panel-action-btn"
					onclick={loadDatabases}
					disabled={isLoading}
					title="Refresh databases"
					data-testid="btn-refresh-databases"
				>
					<Icon name="refresh" size={12} class={isLoading ? 'spinning' : ''} />
				</button>
				<button
					class="panel-action-btn"
					onclick={openCreateModal}
					title="Create database"
					data-testid="btn-create-database"
				>
					<Icon name="plus" size={12} />
				</button>
			</div>
		</div>

		{#if isExpanded}
			<div class="panel-body">
				{#if isLoading && databases.length === 0}
					<div class="panel-status">
						<Icon name="refresh" size={12} class="spinning" />
						Loading...
					</div>
				{:else if error}
					<div class="panel-error">
						<Icon name="alert-circle" size={12} />
						{error}
					</div>
				{:else if databases.length === 0}
					<div class="panel-status">No databases</div>
				{:else}
					{#each databases as db}
						{@const isActive = db.name === $activeConnection.database}
						{@const isBusy = busyDatabase === db.name}
						<div
							class="db-row"
							class:active={isActive}
							class:busy={isBusy}
							data-testid="database-row-{db.name}"
						>
							<button
								class="db-row-button"
								data-testid="database-option-{db.name}"
								onclick={() => handleSwitch(db.name)}
								oncontextmenu={(e) => openContextMenu(e, db.name)}
								disabled={isBusy}
							>
								<span class="db-marker" aria-hidden="true"></span>
								<Icon name={isBusy ? 'refresh' : 'database'} size={12} class={isBusy ? 'spinning' : ''} />
								<span class="db-name">{db.name}</span>
								{#if db.size}
									<span class="db-meta">{db.size}</span>
								{/if}
							</button>
							<button
								class="db-row-kebab"
								onclick={(e) => openKebab(e, db.name)}
								title="More options"
								data-testid="database-kebab-{db.name}"
							>
								<Icon name="dots-vertical" size={12} />
							</button>
						</div>
					{/each}
				{/if}
			</div>
		{/if}
	</div>

	{#if contextMenu}
		{@const menu = contextMenu}
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div class="ctx-backdrop" onclick={closeContextMenu}></div>
		<div
			class="ctx-menu"
			style="left: {menu.x}px; top: {menu.y}px"
			data-testid="database-context-menu"
		>
			<button class="ctx-item" onclick={() => handleCopyName(menu.database)}>
				<Icon name="copy" size={12} />
				Copy name
			</button>
			<div class="ctx-separator"></div>
			<button
				class="ctx-item ctx-item-danger"
				onclick={() => openDropModal(menu.database)}
				data-testid="ctx-drop-database"
			>
				<Icon name="trash" size={12} />
				Drop database...
			</button>
		</div>
	{/if}

	{#if createModal}
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div class="modal-backdrop" onclick={closeCreateModal}></div>
		<div class="modal" data-testid="create-database-modal">
			<div class="modal-header modal-header-primary">
				<Icon name="database" size={18} />
				<h3>Create Database</h3>
			</div>
			<div class="modal-body">
				<label class="form-field">
					<span>Name *</span>
					<input
						type="text"
						bind:value={createModal.name}
						placeholder="my_database"
						data-testid="input-create-db-name"
						onkeydown={(e) => {
							if (e.key === 'Enter') confirmCreate();
						}}
					/>
				</label>
				<label class="form-field">
					<span>Owner (optional)</span>
					<input type="text" bind:value={createModal.owner} placeholder="current user" />
				</label>
				<label class="form-field">
					<span>Template (optional)</span>
					<input type="text" bind:value={createModal.template} placeholder="template0" />
				</label>
				{#if createError}
					<div class="modal-error">
						<Icon name="alert-circle" size={14} />
						{createError}
					</div>
				{/if}
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary btn-sm" onclick={closeCreateModal} disabled={isCreating}>
					Cancel
				</button>
				<button
					class="btn btn-primary btn-sm"
					onclick={confirmCreate}
					disabled={isCreating}
					data-testid="btn-confirm-create-database"
				>
					{#if isCreating}
						<Icon name="refresh" size={14} class="spinning" />
						Creating...
					{:else}
						Create Database
					{/if}
				</button>
			</div>
		</div>
	{/if}

	{#if dropModal}
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div class="modal-backdrop" onclick={closeDropModal}></div>
		<div class="modal" data-testid="drop-database-modal">
			<div class="modal-header">
				<Icon name="alert-circle" size={18} />
				<h3>Drop Database</h3>
			</div>
			<div class="modal-body">
				<p>
					Drop database <strong>"{dropModal.database}"</strong>?
				</p>
				<p class="warning-text">This action cannot be undone.</p>
				{#if dropModal.database === $activeConnection.database}
					<p class="warning-text warning-note">
						This is the currently-selected database. You'll be switched to <code>postgres</code> first.
					</p>
				{/if}
				<label class="cascade-option">
					<input type="checkbox" bind:checked={dropModal.force} data-testid="checkbox-force-drop" />
					<span>Force — terminate active sessions before dropping</span>
				</label>
				{#if dropError}
					<div class="modal-error">
						<Icon name="alert-circle" size={14} />
						{dropError}
					</div>
				{/if}
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary btn-sm" onclick={closeDropModal} disabled={isDropping}>
					Cancel
				</button>
				<button
					class="btn btn-danger btn-sm"
					onclick={confirmDrop}
					disabled={isDropping}
					data-testid="btn-confirm-drop-database"
				>
					{#if isDropping}
						<Icon name="refresh" size={14} class="spinning" />
						Dropping...
					{:else}
						Drop Database
					{/if}
				</button>
			</div>
		</div>
	{/if}
{/if}

<style>
	.databases-panel {
		display: flex;
		flex-direction: column;
		border-bottom: 1px solid var(--color-border);
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 6px 8px 6px 12px;
	}

	.panel-title {
		display: flex;
		align-items: center;
		gap: 6px;
		flex: 1;
		min-width: 0;
		padding: 2px 4px;
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		color: var(--color-text-muted);
		letter-spacing: 0.3px;
		background: transparent;
		border: none;
		cursor: pointer;
		text-align: left;
	}

	.panel-title :global(svg) {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.panel-chevron {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 12px;
		transition: transform var(--transition-fast);
	}

	.panel-chevron.expanded {
		transform: rotate(90deg);
	}

	.panel-count {
		margin-left: auto;
		padding: 1px 6px;
		font-size: 10px;
		background: var(--color-surface);
		color: var(--color-text-dim);
		border-radius: 8px;
		text-transform: none;
		letter-spacing: 0;
	}

	.panel-actions {
		display: flex;
		gap: 2px;
	}

	.panel-action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.panel-action-btn:hover:not(:disabled) {
		background: var(--color-surface);
		color: var(--color-text);
	}

	.panel-action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.panel-body {
		display: flex;
		flex-direction: column;
		padding: 2px 6px 6px;
		max-height: 260px;
		overflow-y: auto;
	}

	.panel-status,
	.panel-error {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 10px;
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.panel-error {
		color: var(--color-error);
	}

	.db-row {
		display: flex;
		align-items: stretch;
		border-radius: var(--radius-sm);
		position: relative;
	}

	.db-row:hover {
		background: var(--color-surface);
	}

	.db-row.active {
		background: rgba(137, 180, 250, 0.1);
	}

	.db-row-button {
		display: flex;
		align-items: center;
		gap: 6px;
		flex: 1;
		min-width: 0;
		padding: 4px 6px;
		font-size: 12px;
		text-align: left;
		background: transparent;
		border: none;
		cursor: pointer;
		color: var(--color-text);
	}

	.db-row-button:disabled {
		cursor: not-allowed;
	}

	.db-row-button :global(svg) {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.db-row.active .db-row-button {
		color: var(--color-primary);
	}

	.db-row.active .db-row-button :global(svg) {
		color: var(--color-primary);
	}

	.db-marker {
		width: 3px;
		min-height: 16px;
		border-radius: 2px;
		background: transparent;
		flex-shrink: 0;
	}

	.db-row.active .db-marker {
		background: var(--color-primary);
	}

	.db-name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.db-meta {
		font-size: 10px;
		color: var(--color-text-dim);
	}

	.db-row-kebab {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		padding: 2px;
		background: transparent;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: var(--radius-sm);
		opacity: 0;
		transition: opacity var(--transition-fast), color var(--transition-fast);
	}

	.db-row:hover .db-row-kebab {
		opacity: 1;
	}

	.db-row-kebab:hover {
		color: var(--color-text);
		background: var(--color-bg-tertiary);
	}

	.ctx-backdrop {
		position: fixed;
		inset: 0;
		z-index: 999;
	}

	.ctx-menu {
		position: fixed;
		z-index: 1000;
		min-width: 180px;
		padding: 4px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
	}

	.ctx-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 6px 10px;
		font-size: 12px;
		text-align: left;
		background: transparent;
		border: none;
		cursor: pointer;
		border-radius: var(--radius-sm);
		color: var(--color-text);
	}

	.ctx-item:hover {
		background: var(--color-surface);
	}

	.ctx-item :global(svg) {
		color: var(--color-text-muted);
	}

	.ctx-separator {
		height: 1px;
		background: var(--color-border);
		margin: 4px 0;
	}

	.ctx-item-danger {
		color: var(--color-error);
	}

	.ctx-item-danger :global(svg) {
		color: var(--color-error);
	}

	.ctx-item-danger:hover {
		background: rgba(243, 139, 168, 0.15);
	}

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
		min-width: 420px;
		max-width: 520px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
	}

	.modal-header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 18px;
		border-bottom: 1px solid var(--color-border);
		color: var(--color-error);
	}

	.modal-header-primary {
		color: var(--color-primary);
	}

	.modal-header h3 {
		margin: 0;
		font-size: 15px;
		font-weight: 600;
		color: var(--color-text);
	}

	.modal-body {
		padding: 18px;
	}

	.modal-body p {
		margin: 0 0 10px;
	}

	.warning-text {
		color: var(--color-error);
		font-weight: 500;
	}

	.warning-note {
		font-weight: 400;
		font-size: 13px;
	}

	.warning-note code {
		font-family: var(--font-mono);
		padding: 1px 4px;
		background: var(--color-surface);
		border-radius: 3px;
	}

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

	.form-field input {
		padding: 8px 10px;
		font-size: 13px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.form-field input:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.cascade-option {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 14px;
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

	.modal-error {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		margin: 12px 0 0;
		background: rgba(243, 139, 168, 0.15);
		border: 1px solid var(--color-error);
		border-radius: var(--radius-sm);
		color: var(--color-error);
		font-size: 13px;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 14px 18px;
		border-top: 1px solid var(--color-border);
	}

	.spinning {
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
