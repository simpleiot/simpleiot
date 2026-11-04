<script lang="ts">
	// A widget shows points from other nodes on a dashboard: its component,
	// its place on the grid, its display options, and a binding per prop
	// the component reads.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import SelectInput from "#lib/inputs/SelectInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getBestDesc, getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { widgets } from "#lib/widgets/index.ts"
	import { autoWidget } from "#lib/widgets/registry.ts"
	import type { EditorProps } from "../types.ts"
	import BindingInputs from "./BindingInputs.svelte"
	import BoxIcon from "@lucide/svelte/icons/box"

	let props: EditorProps = $props()

	const component = $derived(getText(props.node.points, PointType.Component) || autoWidget)
	// an auto widget binds one value; a named one binds what it declares
	const bindings = $derived(widgets.get(component)?.bindings ?? [PointType.Value])
	const optionNames = $derived(Object.keys(widgets.get(component)?.options ?? {}))

	// the dashboard's parent, which a relative binding starts from
	const via = $derived.by(() => {
		const viaId = props.parent?.parent
		if (!viaId) {
			return undefined
		}
		const n = props.findNode(viaId)
		return { id: viaId, description: n ? getBestDesc(n.points) : viaId }
	})
</script>

<NodeEditor {props} icon={BoxIcon}>
	{#snippet header()}
		<span class="component">({component})</span>
	{/snippet}
	<TextInput
		type={PointType.Description}
		label="Label"
		placeholder="the bound node's description"
	/>
	<SelectInput
		type={PointType.Component}
		label="Component"
		options={[
			[autoWidget, "auto (by the node's type)"],
			...widgets.all.map((w): [string, string] => [w.name, w.label]),
		]}
	/>
	{#each bindings as name (name)}
		<BindingInputs {name} {props} {via} />
	{/each}
	<div class="row">
		<NumberInput type={PointType.X} label="X" />
		<NumberInput type={PointType.Y} label="Y" />
		<NumberInput type={PointType.W} label="Width" />
		<NumberInput type={PointType.H} label="Height" />
	</div>
	<div class="hint">Columns and rows on a grid; canvas units on a canvas.</div>
	<KeyValueInput type={PointType.Prop} label="Options" buttonLabel="Add option" />
	{#if optionNames.length > 0}
		<div class="hint">Options for this component: {optionNames.join(", ")}.</div>
	{/if}
	<KeyValueInput type={PointType.Port} label="Ports" buttonLabel="Add port" />
	<div class="hint">
		A port is "x,y" or "x,y,dir" as fractions of the widget; an empty value removes a default.
	</div>
</NodeEditor>

<style>
	.component {
		color: var(--muted);
	}

	.row {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.hint {
		color: var(--muted);
	}
</style>
