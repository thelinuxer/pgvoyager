<script lang="ts">
	import { iconLibrary, type IconData } from './index';
	import { lucideIcons } from './icons/lucide';
	import { heroicons } from './icons/heroicons';
	import { phosphorIcons } from './icons/phosphor';
	import { tablerIcons } from './icons/tabler';

	interface Props {
		name: string;
		size?: number;
		class?: string;
		strokeWidth?: number;
	}

	let { name, size = 16, class: className = '', strokeWidth = 2 }: Props = $props();

	const libraries: Record<string, Record<string, IconData>> = {
		lucide: lucideIcons,
		heroicons: heroicons,
		phosphor: phosphorIcons,
		tabler: tablerIcons
	};

	let iconData = $derived.by(() => {
		const lib = libraries[$iconLibrary];
		return lib?.[name] || lucideIcons[name];
	});
</script>

{#if iconData}
	<svg
		width={size}
		height={size}
		viewBox={iconData.viewBox}
		fill="none"
		stroke="currentColor"
		stroke-width={strokeWidth}
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		{#each iconData.paths as path}
			{#if path.fill}
				<path d={path.d} fill={path.fill} stroke="none" />
			{:else}
				<path d={path.d} stroke-width={path.strokeWidth ?? strokeWidth} />
			{/if}
		{/each}
	</svg>
{:else}
	<!-- Fallback: question mark in box -->
	<svg
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width={strokeWidth}
		class={className}
	>
		<rect x="3" y="3" width="18" height="18" rx="2" />
		<path d="M9 9h.01M12 12v4M12 8a2 2 0 012-2c1.1 0 2 .9 2 2 0 1.1-.9 2-2 2h-2" />
	</svg>
{/if}
