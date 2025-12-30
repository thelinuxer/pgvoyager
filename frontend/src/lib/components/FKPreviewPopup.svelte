<script lang="ts">
	import type { ForeignKeyPreview } from '$lib/types';

	interface Props {
		preview: ForeignKeyPreview | null;
		loading: boolean;
		x: number;
		y: number;
	}

	let { preview, loading, x, y }: Props = $props();

	// Position the popup to avoid going off-screen
	let adjustedX = $derived(Math.min(x + 10, window.innerWidth - 400));
	let adjustedY = $derived(Math.min(y + 10, window.innerHeight - 300));

	function formatValue(value: unknown): string {
		if (value === null) return 'NULL';
		if (value === undefined) return '';
		if (typeof value === 'object') {
			return JSON.stringify(value);
		}
		return String(value);
	}
</script>

<div class="fk-popup" style="left: {adjustedX}px; top: {adjustedY}px">
	{#if loading}
		<div class="fk-loading">Loading...</div>
	{:else if preview}
		<div class="fk-header">
			<span class="fk-table">{preview.schema}.{preview.table}</span>
		</div>
		<div class="fk-content">
			<table class="fk-table-data">
				<tbody>
					{#each preview.columns as col}
						<tr>
							<td class="fk-col-name" class:pk={col.isPrimaryKey}>
								{#if col.isPrimaryKey}
									<svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" class="pk-icon">
										<path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
									</svg>
								{/if}
								{col.name}
							</td>
							<td class="fk-col-value" class:null={preview.row[col.name] === null}>
								{formatValue(preview.row[col.name])}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<div class="fk-hint">Click to open in tab</div>
	{:else}
		<div class="fk-error">Failed to load preview</div>
	{/if}
</div>

<style>
	.fk-popup {
		position: fixed;
		z-index: 1000;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
		min-width: 280px;
		max-width: 400px;
		max-height: 300px;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.fk-header {
		padding: 8px 12px;
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
	}

	.fk-table {
		font-family: var(--font-mono);
		font-size: 12px;
		font-weight: 600;
		color: var(--color-primary);
	}

	.fk-content {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
	}

	.fk-table-data {
		width: 100%;
		font-size: 12px;
		font-family: var(--font-mono);
	}

	.fk-table-data tr:hover {
		background: var(--color-surface);
	}

	.fk-col-name {
		padding: 4px 8px;
		color: var(--color-text-muted);
		white-space: nowrap;
		vertical-align: top;
	}

	.fk-col-name.pk {
		color: var(--color-warning);
	}

	.pk-icon {
		margin-right: 4px;
		vertical-align: middle;
	}

	.fk-col-value {
		padding: 4px 8px;
		word-break: break-all;
		max-width: 200px;
	}

	.fk-col-value.null {
		color: var(--color-text-dim);
		font-style: italic;
	}

	.fk-loading,
	.fk-error {
		padding: 16px;
		text-align: center;
		color: var(--color-text-muted);
	}

	.fk-error {
		color: var(--color-error);
	}

	.fk-hint {
		padding: 6px 12px;
		font-size: 10px;
		color: var(--color-text-dim);
		border-top: 1px solid var(--color-border);
		text-align: center;
	}
</style>
