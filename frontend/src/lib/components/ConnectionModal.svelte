<script lang="ts">
	import { connections, activeConnectionId } from '$lib/stores/connections';
	import { connectionApi } from '$lib/api/client';
	import type { Connection, ConnectionRequest } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		onClose: () => void;
		editConnection?: Connection | null;
	}

	let { onClose, editConnection = null }: Props = $props();

	const isEditMode = $derived(!!editConnection);

	let form = $state<ConnectionRequest>({
		name: editConnection?.name || '',
		host: editConnection?.host || 'localhost',
		port: editConnection?.port || 5432,
		database: editConnection?.database || '',
		username: editConnection?.username || 'postgres',
		password: editConnection?.password || '',
		sslMode: editConnection?.sslMode || 'prefer'
	});

	let isTesting = $state(false);
	let testResult = $state<{ success: boolean; message: string } | null>(null);
	let isSaving = $state(false);
	let isDeleting = $state(false);
	let error = $state<string | null>(null);

	async function handleTest() {
		isTesting = true;
		testResult = null;
		error = null;

		try {
			const result = await connectionApi.test({
				host: form.host,
				port: form.port,
				database: form.database,
				username: form.username,
				password: form.password,
				sslMode: form.sslMode
			});
			testResult = result;
		} catch (e) {
			testResult = {
				success: false,
				message: e instanceof Error ? e.message : 'Connection test failed'
			};
		} finally {
			isTesting = false;
		}
	}

	async function handleSave() {
		if (!form.name || !form.host || !form.database || !form.username) {
			error = 'Please fill in all required fields';
			return;
		}

		isSaving = true;
		error = null;

		try {
			if (isEditMode && editConnection) {
				// Update existing connection
				const conn = await connectionApi.update(editConnection.id, form);
				connections.updateConnection(editConnection.id, conn);
				onClose();
			} else {
				// Create new connection
				const conn = await connectionApi.create(form);
				connections.add(conn);

				// Auto-connect
				await connectionApi.connect(conn.id);
				connections.setConnected(conn.id, true);
				activeConnectionId.set(conn.id);

				onClose();
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save connection';
		} finally {
			isSaving = false;
		}
	}

	async function handleDelete() {
		if (!editConnection) return;

		if (!confirm(`Are you sure you want to delete the connection "${editConnection.name}"?`)) {
			return;
		}

		isDeleting = true;
		error = null;

		try {
			await connectionApi.delete(editConnection.id);
			connections.remove(editConnection.id);

			// If this was the active connection, clear it
			if ($activeConnectionId === editConnection.id) {
				activeConnectionId.set(null);
			}

			onClose();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete connection';
		} finally {
			isDeleting = false;
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="modal-backdrop" onclick={handleBackdropClick}>
	<div class="modal">
		<div class="modal-header">
			<h2>
				<Icon name="layers" size={18} />
				{isEditMode ? 'Edit Connection' : 'New Connection'}
			</h2>
			<button class="modal-close" onclick={onClose} title="Close">
				<Icon name="x" size={18} />
			</button>
		</div>

		<div class="modal-body">
			<div class="form-group">
				<label for="name">Connection Name *</label>
				<input
					type="text"
					id="name"
					bind:value={form.name}
					placeholder="My Database"
				/>
			</div>

			<div class="form-row">
				<div class="form-group flex-2">
					<label for="host">Host *</label>
					<input
						type="text"
						id="host"
						bind:value={form.host}
						placeholder="localhost"
					/>
				</div>
				<div class="form-group flex-1">
					<label for="port">Port *</label>
					<input
						type="number"
						id="port"
						bind:value={form.port}
						placeholder="5432"
					/>
				</div>
			</div>

			<div class="form-group">
				<label for="database">Database *</label>
				<input
					type="text"
					id="database"
					bind:value={form.database}
					placeholder="postgres"
				/>
			</div>

			<div class="form-row">
				<div class="form-group flex-1">
					<label for="username">Username *</label>
					<input
						type="text"
						id="username"
						bind:value={form.username}
						placeholder="postgres"
					/>
				</div>
				<div class="form-group flex-1">
					<label for="password">Password</label>
					<input
						type="password"
						id="password"
						bind:value={form.password}
						placeholder="••••••••"
					/>
				</div>
			</div>

			<div class="form-group">
				<label for="sslMode">SSL Mode</label>
				<select id="sslMode" bind:value={form.sslMode}>
					<option value="disable">Disable</option>
					<option value="prefer">Prefer</option>
					<option value="require">Require</option>
					<option value="verify-ca">Verify CA</option>
					<option value="verify-full">Verify Full</option>
				</select>
			</div>

			{#if testResult}
				<div class="test-result" class:success={testResult.success} class:error={!testResult.success}>
					{#if testResult.success}
						<Icon name="check" size={14} strokeWidth={3} />
					{:else}
						<Icon name="x" size={14} strokeWidth={3} />
					{/if}
					{testResult.message}
				</div>
			{/if}

			{#if error}
				<div class="error-message">{error}</div>
			{/if}
		</div>

		<div class="modal-footer">
			<div class="modal-footer-left">
				<button class="btn btn-secondary" onclick={handleTest} disabled={isTesting}>
					{isTesting ? 'Testing...' : 'Test Connection'}
				</button>
				{#if isEditMode}
					<button class="btn btn-danger" onclick={handleDelete} disabled={isDeleting}>
						{isDeleting ? 'Deleting...' : 'Delete'}
					</button>
				{/if}
			</div>
			<div class="modal-footer-right">
				<button class="btn btn-ghost" onclick={onClose}>Cancel</button>
				<button class="btn btn-primary" onclick={handleSave} disabled={isSaving}>
					{#if isSaving}
						Saving...
					{:else if isEditMode}
						Save Changes
					{:else}
						Save & Connect
					{/if}
				</button>
			</div>
		</div>
	</div>
</div>

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal {
		background: var(--color-bg);
		border-radius: var(--radius-lg);
		box-shadow: 0 16px 64px rgba(0, 0, 0, 0.4);
		width: 100%;
		max-width: 480px;
		max-height: 90vh;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid var(--color-border);
	}

	.modal-header h2 {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 18px;
		font-weight: 600;
	}

	.modal-header h2 svg {
		color: var(--color-primary);
	}

	.modal-close {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		border-radius: var(--radius-sm);
		opacity: 0.5;
		transition: all var(--transition-fast);
	}

	.modal-close:hover {
		opacity: 1;
		background: var(--color-surface);
	}

	.modal-body {
		padding: 20px;
		overflow-y: auto;
	}

	.form-group {
		margin-bottom: 16px;
	}

	.form-group label {
		display: block;
		margin-bottom: 6px;
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text-muted);
	}

	.form-group input,
	.form-group select {
		width: 100%;
	}

	.form-row {
		display: flex;
		gap: 12px;
	}

	.flex-1 {
		flex: 1;
	}

	.flex-2 {
		flex: 2;
	}

	.test-result {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px;
		border-radius: var(--radius-sm);
		margin-top: 16px;
		font-size: 13px;
	}

	.test-result.success {
		background: rgba(166, 227, 161, 0.1);
		color: var(--color-success);
		border: 1px solid var(--color-success);
	}

	.test-result.error {
		background: rgba(243, 139, 168, 0.1);
		color: var(--color-error);
		border: 1px solid var(--color-error);
	}

	.error-message {
		padding: 12px;
		border-radius: var(--radius-sm);
		margin-top: 16px;
		background: rgba(243, 139, 168, 0.1);
		color: var(--color-error);
		border: 1px solid var(--color-error);
		font-size: 13px;
	}

	.modal-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-top: 1px solid var(--color-border);
		background: var(--color-bg-secondary);
	}

	.modal-footer-left,
	.modal-footer-right {
		display: flex;
		gap: 8px;
	}

	:global(.btn-danger) {
		background: var(--color-error);
		color: white;
	}

	:global(.btn-danger:hover) {
		background: color-mix(in srgb, var(--color-error) 85%, black);
	}

	:global(.btn-danger:disabled) {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
