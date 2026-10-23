<script lang="ts">
	// ToolButton is a node operation: an icon with its name beside it, or
	// the icon alone on a narrow screen, where the name is its tooltip.
	import type { Snippet } from "svelte"

	let {
		label,
		onclick,
		pressed,
		children,
	}: {
		label: string
		onclick: (e: MouseEvent) => void
		pressed?: boolean
		children: Snippet
	} = $props()
</script>

<button type="button" class="tool" title={label} aria-pressed={pressed} {onclick}>
	{@render children()}
	<span class="label">{label}</span>
</button>

<style>
	.tool {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px 4px 6px;
		border: 1px solid transparent;
		border-radius: 4px;
		background: none;
		color: inherit;
		font: inherit;
		font-size: 14px;
		cursor: pointer;
	}

	.tool:hover {
		background: var(--hover);
	}

	.tool[aria-pressed="true"] {
		border-color: var(--border);
		background: var(--hover);
	}

	@media (max-width: 600px) {
		.tool {
			padding: 5px;
		}

		.label {
			position: absolute;
			width: 1px;
			height: 1px;
			overflow: hidden;
			clip-path: inset(50%);
			white-space: nowrap;
		}
	}
</style>
