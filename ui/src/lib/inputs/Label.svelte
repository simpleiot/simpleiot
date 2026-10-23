<script lang="ts">
	// Label is the right-aligned label column in front of an input.
	import type { Snippet } from "svelte"
	import { inputContext } from "./context.ts"

	let { label, id, children }: { label: string; id?: string; children: Snippet } = $props()

	const ctx = inputContext()
</script>

<div class="field">
	{#if label !== ""}
		<label for={id} style:width="{ctx.labelWidth}px">{label}:</label>
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
