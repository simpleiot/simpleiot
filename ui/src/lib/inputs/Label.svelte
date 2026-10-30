<script lang="ts">
	// Label is the right-aligned label column in front of an input. A
	// multi-line input sets top so the label lines up with its first line.
	import type { Snippet } from "svelte"
	import { inputContext } from "./context.ts"

	let {
		label,
		id,
		top = false,
		edited = false,
		children,
	}: { label: string; id?: string; top?: boolean; edited?: boolean; children: Snippet } = $props()

	const ctx = inputContext()
</script>

<div class="field" class:top>
	{#if label !== ""}
		<label for={id} class:edited style:width="{ctx.labelWidth}px">{label}:</label>
	{/if}
	{@render children()}
</div>

<style>
	.field {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px 10px;
	}

	.top {
		align-items: baseline;
	}

	/* a field with an unsaved edit */
	.edited {
		color: var(--blue);
		font-weight: 600;
	}

	label {
		text-align: right;
		flex-shrink: 0;
		max-width: 100%;
	}

	@media (max-width: 600px) {
		label {
			text-align: left;
			width: auto !important;
		}
	}
</style>
