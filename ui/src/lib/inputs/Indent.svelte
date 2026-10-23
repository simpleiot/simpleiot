<script lang="ts">
	// Indent lines content up with the inputs, past the label column.
	import type { Snippet } from "svelte"
	import { inputContext } from "./context.ts"

	let { children }: { children: Snippet } = $props()

	const ctx = inputContext()
</script>

<div class="indent" style:--indent="{ctx.labelWidth + 10}px">{@render children()}</div>

<style>
	.indent {
		padding-left: var(--indent);
	}

	@media (max-width: 600px) {
		.indent {
			padding-left: 0;
		}
	}
</style>
