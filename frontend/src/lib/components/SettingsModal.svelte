<script lang="ts">
	import { themeId, themeList, currentTheme } from '$lib/themes';
	import { iconLibrary, iconLibraries, type IconLibrary } from '$lib/icons';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		onClose: () => void;
	}

	let { onClose }: Props = $props();

	function handleThemeChange(id: string) {
		themeId.setTheme(id);
	}

	function handleIconLibraryChange(lib: IconLibrary) {
		iconLibrary.setLibrary(lib);
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-overlay" onclick={handleBackdropClick}>
	<div class="modal">
		<div class="modal-header">
			<h2>Settings</h2>
			<button class="close-btn" onclick={onClose}>
				<Icon name="x" size={18} />
			</button>
		</div>

		<div class="modal-body">
			<section class="settings-section">
				<h3>Theme</h3>
				<div class="theme-grid">
					{#each themeList as theme}
						<button
							class="theme-card"
							class:selected={$themeId === theme.id}
							onclick={() => handleThemeChange(theme.id)}
						>
							<div class="theme-preview" style="background: {theme.colors.bg}">
								<div class="preview-header" style="background: {theme.colors.bgSecondary}; border-color: {theme.colors.border}">
									<span class="preview-dot" style="background: {theme.colors.error}"></span>
									<span class="preview-dot" style="background: {theme.colors.warning}"></span>
									<span class="preview-dot" style="background: {theme.colors.success}"></span>
								</div>
								<div class="preview-content">
									<div class="preview-sidebar" style="background: {theme.colors.bgSecondary}; border-color: {theme.colors.border}"></div>
									<div class="preview-main">
										<div class="preview-line" style="background: {theme.colors.primary}; width: 60%"></div>
										<div class="preview-line" style="background: {theme.colors.textMuted}; width: 80%"></div>
										<div class="preview-line" style="background: {theme.colors.textDim}; width: 40%"></div>
									</div>
								</div>
							</div>
							<div class="theme-info">
								<span class="theme-name">{theme.name}</span>
								<span class="theme-type">{theme.type}</span>
							</div>
							{#if $themeId === theme.id}
								<div class="selected-indicator">
									<Icon name="check" size={14} />
								</div>
							{/if}
						</button>
					{/each}
				</div>
			</section>

			<section class="settings-section">
				<h3>Icon Style</h3>
				<div class="icon-library-grid">
					{#each iconLibraries as lib}
						<button
							class="icon-library-card"
							class:selected={$iconLibrary === lib.id}
							onclick={() => handleIconLibraryChange(lib.id)}
						>
							<div class="icon-preview">
								<Icon name="table" size={20} />
								<Icon name="search" size={20} />
								<Icon name="settings" size={20} />
								<Icon name="folder" size={20} />
							</div>
							<span class="library-name">{lib.name}</span>
							{#if $iconLibrary === lib.id}
								<div class="selected-indicator">
									<Icon name="check" size={14} />
								</div>
							{/if}
						</button>
					{/each}
				</div>
			</section>
		</div>
	</div>
</div>

<style>
	.modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		backdrop-filter: blur(2px);
	}

	.modal {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		width: 600px;
		max-width: 90vw;
		max-height: 85vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 16px 48px rgba(0, 0, 0, 0.3);
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid var(--color-border);
	}

	.modal-header h2 {
		font-size: 18px;
		font-weight: 600;
		margin: 0;
	}

	.close-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 6px;
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}

	.close-btn:hover {
		background: var(--color-surface);
		color: var(--color-text);
	}

	.modal-body {
		flex: 1;
		overflow-y: auto;
		padding: 20px;
	}

	.settings-section {
		margin-bottom: 28px;
	}

	.settings-section:last-child {
		margin-bottom: 0;
	}

	.settings-section h3 {
		font-size: 14px;
		font-weight: 600;
		color: var(--color-text-muted);
		margin: 0 0 12px 0;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.theme-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
		gap: 12px;
	}

	.theme-card {
		position: relative;
		display: flex;
		flex-direction: column;
		border: 2px solid var(--color-border);
		border-radius: var(--radius-md);
		overflow: hidden;
		transition: all var(--transition-fast);
		text-align: left;
	}

	.theme-card:hover {
		border-color: var(--color-text-muted);
	}

	.theme-card.selected {
		border-color: var(--color-primary);
	}

	.theme-preview {
		height: 80px;
		padding: 6px;
		display: flex;
		flex-direction: column;
	}

	.preview-header {
		display: flex;
		gap: 4px;
		padding: 4px 6px;
		border-radius: 3px;
		border-bottom: 1px solid;
		margin-bottom: 4px;
	}

	.preview-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
	}

	.preview-content {
		flex: 1;
		display: flex;
		gap: 4px;
	}

	.preview-sidebar {
		width: 30%;
		border-radius: 2px;
		border-right: 1px solid;
	}

	.preview-main {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 4px;
	}

	.preview-line {
		height: 4px;
		border-radius: 2px;
	}

	.theme-info {
		padding: 8px 10px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: var(--color-bg-secondary);
	}

	.theme-name {
		font-size: 12px;
		font-weight: 500;
	}

	.theme-type {
		font-size: 10px;
		color: var(--color-text-dim);
		text-transform: capitalize;
	}

	.selected-indicator {
		position: absolute;
		top: 6px;
		right: 6px;
		width: 22px;
		height: 22px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--color-primary);
		color: var(--color-bg);
		border-radius: 50%;
	}

	.icon-library-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
		gap: 12px;
	}

	.icon-library-card {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 16px 12px;
		border: 2px solid var(--color-border);
		border-radius: var(--radius-md);
		transition: all var(--transition-fast);
	}

	.icon-library-card:hover {
		border-color: var(--color-text-muted);
	}

	.icon-library-card.selected {
		border-color: var(--color-primary);
	}

	.icon-preview {
		display: flex;
		gap: 8px;
		margin-bottom: 10px;
		color: var(--color-text-muted);
	}

	.icon-library-card.selected .icon-preview {
		color: var(--color-primary);
	}

	.library-name {
		font-size: 12px;
		font-weight: 500;
	}
</style>
