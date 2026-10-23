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
		tone,
		highlight = false,
		showDisabled = true,
		lead,
		children,
	}: {
		node: Node
		icon: Icon
		// description replaces the node's description point
		description?: string
		// tone shades the line: muted for a node that is off or out of
		// touch, error for one reporting an error
		tone?: "muted" | "error" | undefined
		// highlight marks the description, such as for an active rule
		highlight?: boolean
		showDisabled?: boolean
		// lead goes between the icon and the description
		lead?: Snippet
		// children follow the description on the same line
		children?: Snippet
	} = $props()

	const disabled = $derived(showDisabled && getBool(node.points, PointType.Disabled))
</script>

<div class="header" class:muted={tone === "muted"} class:error={tone === "error"}>
	<span class="icon"><IconComponent size={20} /></span>
	{@render lead?.()}
	<span class="desc" class:highlight
		>{description ?? getText(node.points, PointType.Description)}</span
	>
	{@render children?.()}
	{#if disabled}<span>(disabled)</span>{/if}
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

	.muted,
	.error {
		padding-right: 10px;
	}

	.muted {
		background: var(--hover);
	}

	.error {
		background: var(--error-bg);
	}

	.desc {
		overflow-wrap: anywhere;
		white-space: pre-wrap;
	}

	.highlight {
		background: var(--blue);
		color: var(--on-color);
	}
</style>
