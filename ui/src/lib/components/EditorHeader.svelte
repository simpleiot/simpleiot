<script lang="ts">
	// EditorHeader is the line every node editor starts with: the type's
	// icon, the node's description, and a note when it is disabled.
	import type { Snippet } from "svelte"
	import type { Icon } from "#lib/nodes/registry.ts"
	import type { Node } from "#lib/siot/node.ts"
	import { getBool, getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"

	let {
		node,
		icon: IconComponent,
		description,
		children,
	}: {
		node: Node
		icon: Icon
		// description replaces the node's description point
		description?: string
		// children follow the description on the same line
		children?: Snippet
	} = $props()

	const disabled = $derived(getBool(node.points, PointType.Disabled))
</script>

<div class="header">
	<span class="icon"><IconComponent size={20} /></span>
	<span class="desc">{description ?? getText(node.points, PointType.Description)}</span>
	{#if disabled}<span>(disabled)</span>{/if}
	{@render children?.()}
</div>

<style>
	.header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px 10px;
		min-height: 34px;
	}

	.icon {
		display: flex;
		padding: 5px;
	}

	.desc {
		overflow-wrap: anywhere;
	}
</style>
