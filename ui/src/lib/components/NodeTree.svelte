<script lang="ts">
	// NodeTree shows the forest, one tree per group the user belongs to.
	import { store } from "#lib/siot/store.ts"
	import NodeRow from "./NodeRow.svelte"
</script>

<ul class="tree" role="tree" aria-label="Nodes">
	{#each store.forest as t (t.key)}
		<NodeRow {t} parent={undefined} />
	{/each}
</ul>

<style>
	.tree {
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}
</style>
