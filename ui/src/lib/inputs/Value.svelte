<script lang="ts">
	// Value shows a read-only value in the input column, labeled like an
	// input.
	import type { Snippet } from "svelte"
	import Label from "./Label.svelte"

	let { label, children }: { label: string; children: Snippet } = $props()
</script>

<Label {label}><span class="value">{@render children()}</span></Label>

<style>
	.value {
		overflow-wrap: anywhere;
	}
</style>
