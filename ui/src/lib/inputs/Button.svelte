<script lang="ts">
	// Button is a form button in one of the theme colors.
	import type { Snippet } from "svelte"
	import type { Color } from "./color.ts"

	let {
		color = "blue",
		onclick,
		disabled = false,
		children,
	}: { color?: Color; onclick: () => void; disabled?: boolean; children: Snippet } = $props()
</script>

<button type="button" class="btn {color}" {disabled} {onclick}>{@render children()}</button>
