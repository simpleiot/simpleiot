<script lang="ts">
	// The provisioning node gathers the files this instance is configured
	// from: provisioningFile nodes recording what was applied from the
	// provisioning directory, and file nodes uploaded through this UI.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { NodeType, PointType } from "#lib/siot/schema.ts"
	import type { TreeNode } from "#lib/siot/tree.ts"
	import type { EditorProps } from "../types.ts"
	import ListIcon from "@lucide/svelte/icons/list"

	let props: EditorProps = $props()

	// isSource picks out the children that describe a provisioning file, so
	// that anything else living under this node is not counted.
	const isSource = (c: TreeNode) =>
		c.node.type === NodeType.ProvisioningFile || c.node.type === NodeType.File

	const hasError = (c: TreeNode) => getText(c.node.points, PointType.Error) !== ""

	const sources = $derived(props.children.filter(isSource))
	const failed = $derived(sources.filter(hasError).length)

	function summary(count: number): string {
		switch (count) {
			case 0:
				return "no files"
			case 1:
				return "1 file"
			default:
				return `${count} files`
		}
	}
</script>

<NodeEditor {props} icon={ListIcon} description="Provisioning" showDisabled={false}>
	{#snippet header()}
		<span>{summary(sources.length)}</span>
		<!-- a file that did not apply is visible without expanding anything,
		which is the question this node exists to answer -->
		{#if failed > 0}
			<span class="failed">{failed === 1 ? "1 with an error" : `${failed} with errors`}</span>
		{/if}
	{/snippet}
</NodeEditor>

<style>
	.failed {
		color: var(--red);
	}
</style>
