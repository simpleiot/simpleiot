<script lang="ts">
	// A dashboard is a layout of widgets; the tree edits its settings and
	// opens it, and the dashboard page shows it. The open link carries the
	// parent it sits under here, so a mirrored dashboard shows this
	// place's nodes.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import LayoutDashboardIcon from "@lucide/svelte/icons/layout-dashboard"
	import { dashHref } from "./href.ts"

	let props: EditorProps = $props()

	const layout = $derived(getText(props.node.points, PointType.Layout) || PointValue.Grid)
</script>

<NodeEditor {props} icon={LayoutDashboardIcon}>
	{#snippet header()}
		<!-- the path is resolved in dashHref; only the query is appended -->
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		<a class="open" href={dashHref(props.node.id, props.parent?.id ?? "")}>open</a>
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<OptionInput
		type={PointType.Layout}
		label="Layout"
		options={[
			[PointValue.Grid, "grid"],
			[PointValue.Canvas, "canvas"],
		]}
	/>
	{#if layout === PointValue.Canvas}
		<NumberInput type={PointType.CanvasWidth} label="Canvas width" />
		<NumberInput type={PointType.CanvasHeight} label="Canvas height" />
	{:else}
		<NumberInput type={PointType.Cols} label="Columns" />
	{/if}
	<div class="hint">Add widgets under this node, then open the dashboard to see them.</div>
</NodeEditor>

<style>
	.open {
		padding: 2px 10px;
		border-radius: 4px;
		background: var(--badge-bg);
		color: var(--badge-fg);
		text-decoration: none;
	}

	.hint {
		color: var(--muted);
	}
</style>
