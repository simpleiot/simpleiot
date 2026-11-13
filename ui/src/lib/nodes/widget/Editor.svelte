<script lang="ts">
	// A widget shows points from other nodes on a dashboard: its component,
	// its place on the grid, its display options, and a binding per prop
	// the component reads.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import SelectInput from "#lib/inputs/SelectInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import type { Node } from "#lib/siot/node.ts"
	import { getAll, getBestDesc, getText, newText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { widgets } from "#lib/widgets/index.ts"
	import { autoWidget, navigateBinding } from "#lib/widgets/registry.ts"
	import type { EditorProps } from "../types.ts"
	import BindingInputs from "./BindingInputs.svelte"
	import BoxIcon from "@lucide/svelte/icons/box"

	let props: EditorProps = $props()

	const component = $derived(getText(props.node.points, PointType.Component) || autoWidget)
	const descriptor = $derived(widgets.get(component))
	const optionNames = $derived(Object.keys(descriptor?.options ?? {}))

	// a widget that draws a document declares further bindings in it; the
	// document is read from the bound file node, fetched when it is not on
	// the page
	const source = $derived(descriptor?.source)
	const sourceId = $derived(source ? getText(props.node.points, PointType.BindNode, source) : "")
	const sourcePath = $derived(source ? getText(props.node.points, PointType.BindPath, source) : "")
	let fetched = $state<{ id: string; node: Node | undefined } | null>(null)
	$effect(() => {
		const id = sourceId
		if (id === "" || props.findNode(id) || fetched?.id === id) {
			return
		}
		void props.resolveNode(id).then((node) => {
			fetched = { id, node }
		})
	})
	const sourceNode = $derived(
		sourceId === ""
			? undefined
			: (props.findNode(sourceId) ?? (fetched?.id === sourceId ? fetched.node : undefined))
	)
	const declared = $derived(
		sourceNode && descriptor?.bindingsIn
			? descriptor.bindingsIn(getText(sourceNode.points, PointType.Data))
			: []
	)
	const useOf = (name: string) => declared.find((b) => b.name === name)?.use

	// what the editor can say about the document's bindings
	const sourceHint = $derived.by(() => {
		if (!source) {
			return ""
		}
		if (sourcePath !== "") {
			return "Bound by path, so the drawing cannot be read here; add its bindings by name below."
		}
		if (sourceId === "") {
			return "Pick the file holding the drawing; the bindings its elements declare with data-siot-bind are then listed here."
		}
		if (sourceNode && declared.length === 0) {
			return "The drawing declares no bindings; mark the elements that follow a point with data-siot-bind."
		}
		return ""
	})

	// an auto widget binds one value; a named one binds what it declares,
	// what its document declares, and what the node already carries, so a
	// binding the document no longer names is still there to change. Every
	// widget may also name a dashboard a click on it opens.
	const bindings = $derived.by(() => {
		if (!descriptor) {
			return [PointType.Value, navigateBinding]
		}
		const carried = getAll(props.node.points, PointType.BindNode)
			.concat(getAll(props.node.points, PointType.BindPath))
			.filter((p) => p.tombstone === 0 && p.text !== "")
			.map((p) => p.key)
		return [
			...new Set([
				...descriptor.bindings,
				...declared.map((b) => b.name),
				...carried,
				navigateBinding,
			]),
		]
	})

	// a binding added by name, for a drawing bound by a relative path,
	// whose document the editor cannot read
	let added = $state("")
	function addBinding() {
		const name = added.trim()
		added = ""
		if (name !== "" && !bindings.includes(name)) {
			props.onEditPoints([
				newText(PointType.BindType, name, PointType.Value, new Date(props.now).toISOString()),
			])
		}
	}

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
		{@const use = useOf(name)}
		{#if use !== undefined}
			<div class="declared">
				<code>data-siot-bind="{name}"</code>
				<span class="use">{use}</span>
			</div>
		{/if}
		<BindingInputs
			{name}
			{props}
			{via}
			source={name === source || descriptor?.bindingPoints?.[name] === PointType.Data}
			dashboard={name === "dashboard" ||
				name === navigateBinding ||
				(use ?? "").split(", ").includes("open")}
			pointType={descriptor?.bindingPoints?.[name] ?? PointType.Value}
		/>
		{#if name === source && sourceHint !== ""}
			<div class="hint">{sourceHint}</div>
		{/if}
		{#if name === navigateBinding}
			<div class="hint">
				A click on the widget opens this dashboard, except where the widget acts on the click
				itself.
			</div>
		{/if}
	{/each}
	{#if source}
		<div class="add">
			<input
				type="text"
				placeholder="binding name"
				bind:value={added}
				onkeydown={(e) => {
					if (e.key === "Enter") {
						e.preventDefault()
						addBinding()
					}
				}}
			/>
			<button class="btn gray" type="button" onclick={addBinding}>Add binding</button>
		</div>
	{/if}
	<div class="row">
		<NumberInput type={PointType.X} label="X" />
		<NumberInput type={PointType.Y} label="Y" />
		<NumberInput type={PointType.W} label="Width" />
		<NumberInput type={PointType.H} label="Height" />
		<NumberInput type={PointType.Z} label="Z" />
	</div>
	<div class="hint">
		Columns and rows on a grid; canvas units on a canvas. On a canvas a higher Z draws on top.
	</div>
	<KeyValueInput type={PointType.Prop} label="Options" buttonLabel="Add option" />
	<div class="hint">
		Every widget takes <code>scale</code> (1 is normal size) and <code>background</code>
		(<code>on</code> or a color, for a card behind it), and <code>url</code> (an address a click
		opens in a new tab; <code>:3000/path</code> means port 3000 on this host){optionNames.length > 0
			? `; this component also takes ${optionNames.join(", ")}`
			: ""}.
	</div>
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

	.add {
		display: flex;
		gap: 10px;
	}

	/* a binding the drawing declares, named as the drawing names it */
	.declared {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 10px;
		margin-top: 8px;
	}

	.use {
		color: var(--muted);
		font-size: 13px;
	}
</style>
