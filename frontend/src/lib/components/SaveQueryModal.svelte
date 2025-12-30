<script lang="ts">
	import { savedQueries } from '$lib/stores/savedQueries';
	import { activeConnectionId, activeConnection } from '$lib/stores/connections';
	import type { SavedQuery } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		sql: string;
		editQuery?: SavedQuery | null;
		onClose: () => void;
		onSaved?: (query: SavedQuery) => void;
	}

	let { sql, editQuery = null, onClose, onSaved }: Props = $props();

	const isEditMode = $derived(!!editQuery);

	let name = $state(editQuery?.name || '');
	let description = $state(editQuery?.description || '');
	let bindToConnection = $state(!!editQuery?.connectionId);
	let isSaving = $state(false);
	let error = $state<string | null>(null);

	async function handleSave() {
		if (!name.trim()) {
			error = 'Please enter a name';
			return;
		}

		isSaving = true;
		error = null;

		try {
			const data = {
				name: name.trim(),
				sql,
				connectionId: bindToConnection ? $activeConnectionId || undefined : undefined,
				description: description.trim() || undefined
			};

			let savedQuery: SavedQuery;
			if (isEditMode && editQuery) {
				savedQuery = await savedQueries.update(editQuery.id, data);
			} else {
				savedQuery = await savedQueries.add(data);
			}

			onSaved?.(savedQuery);
			onClose();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save query';
		} finally {
			isSaving = false;
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
			e.preventDefault();
			handleSave();
		}
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="modal-backdrop" onclick={handleBackdropClick} onkeydown={handleKeydown}>
	<div class="modal">
		<div class="modal-header">
			<h2>
				<Icon name="save" size={18} />
				{isEditMode ? 'Edit Saved Query' : 'Save Query'}
			</h2>
			<button class="modal-close" onclick={onClose} title="Close">
				<Icon name="x" size={18} />
			</button>
		</div>

		<div class="modal-body">
			<div class="form-group">
				<label for="name">Name *</label>
				<input
					type="text"
					id="name"
					bind:value={name}
					placeholder="My Query"
					autofocus
				/>
			</div>

			<div class="form-group">
				<label for="description">Description</label>
				<textarea
					id="description"
					bind:value={description}
					placeholder="Optional description..."
					rows="2"
				></textarea>
			</div>

			<div class="form-group">
				<label class="checkbox-label">
					<input
						type="checkbox"
						bind:checked={bindToConnection}
						disabled={!$activeConnectionId}
					/>
					<span>Bind to current connection ({$activeConnection?.name || 'None'})</span>
				</label>
				<p class="form-hint">When bound, this query will only appear when connected to this database</p>
			</div>

			<div class="sql-preview">
				<label>SQL Preview</label>
				<pre>{sql.length > 500 ? sql.substring(0, 500) + '...' : sql}</pre>
			</div>

			{#if error}
				<div class="error-message">{error}</div>
			{/if}
		</div>

		<div class="modal-footer">
			<button class="btn btn-ghost" onclick={onClose}>Cancel</button>
			<button class="btn btn-primary" onclick={handleSave} disabled={isSaving}>
				{#if isSaving}
					Saving...
				{:else if isEditMode}
					Save Changes
				{:else}
					Save Query
				{/if}
			</button>
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
		max-width: 500px;
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

	.form-group input[type="text"],
	.form-group textarea {
		width: 100%;
	}

	.form-group textarea {
		resize: vertical;
		min-height: 60px;
	}

	.checkbox-label {
		display: flex !important;
		align-items: center;
		gap: 8px;
		cursor: pointer;
	}

	.checkbox-label input[type="checkbox"] {
		width: auto;
	}

	.checkbox-label span {
		font-size: 14px;
		color: var(--color-text);
	}

	.form-hint {
		font-size: 12px;
		color: var(--color-text-dim);
		margin-top: 4px;
	}

	.sql-preview {
		margin-top: 16px;
	}

	.sql-preview label {
		display: block;
		margin-bottom: 6px;
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text-muted);
	}

	.sql-preview pre {
		background: var(--color-bg-tertiary);
		padding: 12px;
		border-radius: var(--radius-sm);
		font-family: var(--font-mono);
		font-size: 12px;
		overflow-x: auto;
		max-height: 150px;
		white-space: pre-wrap;
		word-break: break-all;
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
		justify-content: flex-end;
		gap: 8px;
		padding: 16px 20px;
		border-top: 1px solid var(--color-border);
		background: var(--color-bg-secondary);
	}
</style>
