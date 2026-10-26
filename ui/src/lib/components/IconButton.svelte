<script lang="ts">
	// IconButton is a round button holding an icon, named for screen
	// readers by its label.
	import type { Snippet } from "svelte"

	let {
		label,
		onclick,
		pressed,
		disabled = false,
		children,
	}: {
		label: string
		onclick: (e: MouseEvent) => void
		pressed?: boolean
		disabled?: boolean
		children: Snippet
	} = $props()
</script>

<button
	type="button"
	class="icon-button"
	title={label}
	aria-label={label}
	aria-pressed={pressed}
	{disabled}
	{onclick}
>
	{@render children()}
</button>

<style>
	.icon-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 5px;
		border: none;
		border-radius: 50%;
		background: none;
		cursor: pointer;
		color: inherit;
	}

	.icon-button:hover:not(:disabled) {
		background: var(--hover);
	}

	.icon-button:disabled {
		opacity: 0.4;
		cursor: default;
	}
</style>
