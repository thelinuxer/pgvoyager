<script lang="ts">
	interface Props {
		direction: 'horizontal' | 'vertical';
		onResize: (delta: number) => void;
	}

	let { direction, onResize }: Props = $props();

	let isDragging = $state(false);
	let startPos = $state(0);

	function handleMouseDown(e: MouseEvent) {
		e.preventDefault();
		isDragging = true;
		startPos = direction === 'horizontal' ? e.clientX : e.clientY;

		document.addEventListener('mousemove', handleMouseMove);
		document.addEventListener('mouseup', handleMouseUp);
		document.body.style.cursor = direction === 'horizontal' ? 'col-resize' : 'row-resize';
		document.body.style.userSelect = 'none';
	}

	function handleMouseMove(e: MouseEvent) {
		if (!isDragging) return;

		const currentPos = direction === 'horizontal' ? e.clientX : e.clientY;
		const delta = currentPos - startPos;
		startPos = currentPos;

		onResize(delta);
	}

	function handleMouseUp() {
		isDragging = false;
		document.removeEventListener('mousemove', handleMouseMove);
		document.removeEventListener('mouseup', handleMouseUp);
		document.body.style.cursor = '';
		document.body.style.userSelect = '';
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="resize-handle"
	class:horizontal={direction === 'horizontal'}
	class:vertical={direction === 'vertical'}
	class:dragging={isDragging}
	onmousedown={handleMouseDown}
>
	<div class="resize-line"></div>
</div>

<style>
	.resize-handle {
		position: relative;
		flex-shrink: 0;
		z-index: 10;
		background: transparent;
	}

	.resize-handle.horizontal {
		width: 5px;
		cursor: col-resize;
	}

	.resize-handle.vertical {
		height: 5px;
		cursor: row-resize;
	}

	.resize-line {
		position: absolute;
		background: var(--color-border);
		transition: background var(--transition-fast);
	}

	.resize-handle.horizontal .resize-line {
		top: 0;
		bottom: 0;
		left: 0;
		width: 1px;
	}

	.resize-handle.vertical .resize-line {
		left: 0;
		right: 0;
		top: 2px;
		height: 1px;
	}

	.resize-handle:hover .resize-line,
	.resize-handle.dragging .resize-line {
		background: var(--color-primary);
	}

	.resize-handle.horizontal:hover .resize-line,
	.resize-handle.horizontal.dragging .resize-line {
		width: 3px;
		left: 1px;
	}

	.resize-handle.vertical:hover .resize-line,
	.resize-handle.vertical.dragging .resize-line {
		height: 3px;
		top: 1px;
	}
</style>
