<script lang="ts">
	// A deep link: the tree, opened down to one node.
	import { page } from "$app/state"
	import TreePage from "#lib/components/TreePage.svelte"
</script>

<TreePage deepLink={page.params.id} />
