<script lang="ts">
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		value: unknown;
		dataType: string;
		columnName: string;
		x: number;
		y: number;
		onClose: () => void;
	}

	let { value, dataType, columnName, x, y, onClose }: Props = $props();

	// Popup dimensions (resizable)
	let width = $state(500);
	let height = $state(350);

	// Position state (draggable)
	let posX = $state(Math.min(Math.max(x - 250, 10), window.innerWidth - 520));
	let posY = $state(Math.min(Math.max(y - 100, 10), window.innerHeight - 370));

	// Drag state
	let isDragging = $state(false);
	let dragStartX = $state(0);
	let dragStartY = $state(0);
	let dragStartPosX = $state(0);
	let dragStartPosY = $state(0);

	// Resize state
	let isResizing = $state(false);
	let resizeStartX = $state(0);
	let resizeStartY = $state(0);
	let resizeStartW = $state(0);
	let resizeStartH = $state(0);

	function formatContent(val: unknown, type: string): string {
		if (val === null || val === undefined) return 'NULL';
		const lowerType = type.toLowerCase();
		if (lowerType === 'json' || lowerType === 'jsonb' || lowerType.includes('json')) {
			try {
				if (typeof val === 'object') return JSON.stringify(val, null, 2);
				const parsed = JSON.parse(String(val));
				return JSON.stringify(parsed, null, 2);
			} catch {
				return String(val);
			}
		}
		if (lowerType === 'xml') return formatXml(String(val));
		if (typeof val === 'object') return JSON.stringify(val, null, 2);
		return String(val);
	}

	function formatXml(xml: string): string {
		let formatted = '';
		let indent = 0;
		const tab = '  ';
		xml = xml.replace(/>\s*</g, '><');
		xml.split(/(<[^>]+>)/g).forEach((node) => {
			if (!node.trim()) return;
			if (node.match(/^<\/\w/)) indent = Math.max(0, indent - 1);
			formatted += tab.repeat(indent) + node + '\n';
			if (node.match(/^<\w[^>]*[^/]>$/)) indent++;
		});
		return formatted.trim();
	}

	// Syntax highlighting
	function escapeHtml(str: string): string {
		return str
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;');
	}

	function highlightJson(text: string): string {
		return text.replace(
			/("(?:[^"\\]|\\.)*")(\s*:)?|(\b(?:true|false|null)\b)|(-?\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b)/g,
			(match, str, colon, keyword, num) => {
				if (str && colon) {
					// Key
					return `<span class="hl-key">${escapeHtml(str)}</span>${colon}`;
				}
				if (str) {
					// String value
					return `<span class="hl-string">${escapeHtml(str)}</span>`;
				}
				if (keyword) {
					return `<span class="hl-keyword">${keyword}</span>`;
				}
				if (num) {
					return `<span class="hl-number">${num}</span>`;
				}
				return match;
			}
		);
	}

	function highlightXml(text: string): string {
		return text.replace(
			/(<\/?)(\w[\w.-]*)([^>]*?)(\/?>)|([^<]+)/g,
			(match, open, tag, attrs, close, textContent) => {
				if (tag) {
					// Highlight attributes within the tag
					const highlightedAttrs = attrs.replace(
						/([\w.-]+)(\s*=\s*)("[^"]*"|'[^']*')/g,
						(_m: string, name: string, eq: string, val: string) =>
							`<span class="hl-attr-name">${escapeHtml(name)}</span>${eq}<span class="hl-attr-value">${escapeHtml(val)}</span>`
					);
					return `${escapeHtml(open)}<span class="hl-tag">${escapeHtml(tag)}</span>${highlightedAttrs}${escapeHtml(close)}`;
				}
				if (textContent) {
					return `<span class="hl-text">${escapeHtml(textContent)}</span>`;
				}
				return escapeHtml(match);
			}
		);
	}

	let formattedContent = $derived(formatContent(value, dataType));
	let isJson = $derived(dataType.toLowerCase().includes('json'));
	let isXml = $derived(dataType.toLowerCase() === 'xml');

	let highlightedHtml = $derived.by(() => {
		if (isJson) return highlightJson(formattedContent);
		if (isXml) return highlightXml(formattedContent);
		return escapeHtml(formattedContent);
	});

	function handleCopy() {
		navigator.clipboard.writeText(formattedContent);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onClose();
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) onClose();
	}

	// Drag handlers
	function handleDragStart(e: MouseEvent) {
		if ((e.target as HTMLElement).closest('.popup-btn')) return;
		isDragging = true;
		dragStartX = e.clientX;
		dragStartY = e.clientY;
		dragStartPosX = posX;
		dragStartPosY = posY;
		e.preventDefault();
	}

	// Resize handlers
	function handleResizeStart(e: MouseEvent) {
		isResizing = true;
		resizeStartX = e.clientX;
		resizeStartY = e.clientY;
		resizeStartW = width;
		resizeStartH = height;
		e.preventDefault();
		e.stopPropagation();
	}

	function handleMouseMove(e: MouseEvent) {
		if (isDragging) {
			posX = dragStartPosX + (e.clientX - dragStartX);
			posY = dragStartPosY + (e.clientY - dragStartY);
		} else if (isResizing) {
			width = Math.max(300, resizeStartW + (e.clientX - resizeStartX));
			height = Math.max(200, resizeStartH + (e.clientY - resizeStartY));
		}
	}

	function handleMouseUp() {
		isDragging = false;
		isResizing = false;
	}
</script>

<svelte:window onkeydown={handleKeydown} onmousemove={handleMouseMove} onmouseup={handleMouseUp} />

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="popup-backdrop" onclick={handleBackdropClick}>
	<div
		class="data-popup"
		style="left: {posX}px; top: {posY}px; width: {width}px; height: {height}px"
	>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="popup-header" onmousedown={handleDragStart} class:dragging={isDragging}>
			<div class="popup-title">
				<span class="column-name">{columnName}</span>
				<span class="data-type">{dataType}</span>
			</div>
			<div class="popup-actions">
				<button class="popup-btn" onclick={handleCopy} title="Copy to clipboard">
					<Icon name="copy" size={14} />
				</button>
				<button class="popup-btn" onclick={onClose} title="Close (Esc)">
					<Icon name="x" size={14} />
				</button>
			</div>
		</div>
		<div class="popup-content">
			<pre>{@html highlightedHtml}</pre>
		</div>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="resize-handle" onmousedown={handleResizeStart}>
			<svg width="10" height="10" viewBox="0 0 10 10">
				<path d="M8 2L2 8M8 5L5 8M8 8L8 8" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
			</svg>
		</div>
	</div>
</div>

<style>
	.popup-backdrop {
		position: fixed;
		inset: 0;
		z-index: 999;
		background: rgba(0, 0, 0, 0.3);
	}

	.data-popup {
		position: fixed;
		z-index: 1000;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.popup-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 12px;
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
		cursor: grab;
		user-select: none;
	}

	.popup-header.dragging {
		cursor: grabbing;
	}

	.popup-title {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.column-name {
		font-family: var(--font-mono);
		font-size: 13px;
		font-weight: 600;
		color: var(--color-text);
	}

	.data-type {
		font-size: 11px;
		color: var(--color-text-muted);
		padding: 2px 6px;
		background: var(--color-bg);
		border-radius: 4px;
	}

	.popup-actions {
		display: flex;
		gap: 4px;
	}

	.popup-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		border-radius: 4px;
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
		cursor: pointer;
	}

	.popup-btn:hover {
		background: var(--color-bg);
		color: var(--color-text);
	}

	.popup-content {
		flex: 1;
		overflow: auto;
		padding: 12px;
	}

	pre {
		margin: 0;
		font-family: var(--font-mono);
		font-size: 12px;
		line-height: 1.6;
		white-space: pre-wrap;
		word-wrap: break-word;
		color: var(--color-text);
	}

	/* Syntax highlighting - JSON */
	pre :global(.hl-key) {
		color: #89b4fa;
	}

	pre :global(.hl-string) {
		color: #a6e3a1;
	}

	pre :global(.hl-number) {
		color: #fab387;
	}

	pre :global(.hl-keyword) {
		color: #cba6f7;
	}

	/* Syntax highlighting - XML */
	pre :global(.hl-tag) {
		color: #f38ba8;
	}

	pre :global(.hl-attr-name) {
		color: #fab387;
	}

	pre :global(.hl-attr-value) {
		color: #a6e3a1;
	}

	pre :global(.hl-text) {
		color: var(--color-text);
	}

	/* Resize handle */
	.resize-handle {
		position: absolute;
		bottom: 0;
		right: 0;
		width: 20px;
		height: 20px;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: nwse-resize;
		color: var(--color-text-dim);
		opacity: 0.6;
		transition: opacity var(--transition-fast);
	}

	.resize-handle:hover {
		opacity: 1;
	}
</style>
