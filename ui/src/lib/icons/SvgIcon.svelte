<script lang="ts">
	// The frame of a custom icon, drawn the way Lucide draws its icons so
	// the two sets match.
	import type { Snippet } from "svelte"

	let { size = 24, children }: { size?: number | string; children: Snippet } = $props()
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="2"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	{@render children()}
</svg>
