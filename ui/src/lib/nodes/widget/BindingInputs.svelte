<script lang="ts">
	// BindingInputs edits one binding of a widget: the node, through the
	// picker, then the point type and key, offered from what the chosen
	// node carries.
	import NodeRefInput from "#lib/inputs/NodeRefInput.svelte"
	import SelectInput from "#lib/inputs/SelectInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { pointKeys, pointTypes, type Node } from "#lib/siot/node.ts"
	import { getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"

	let { name, props }: { name: string; props: EditorProps } = $props()

	const points = $derived(props.node.points)
	const nodeId = $derived(getText(points, PointType.BindNode, name))
	const type = $derived(getText(points, PointType.BindType, name) || PointType.Value)

	let resolved = $state<{ id: string; node: Node | undefined } | null>(null)
	const onPage = $derived(nodeId === "" ? undefined : props.findNode(nodeId))
	$effect(() => {
		const id = nodeId
		if (id === "" || onPage || resolved?.id === id) {
			return
		}
		void props.resolveNode(id).then((node) => {
			resolved = { id, node }
		})
	})
	const target = $derived(onPage ?? (resolved?.id === nodeId ? resolved.node : undefined))
</script>

<NodeRefInput
	type={PointType.BindNode}
	key={name}
	label="{name} node"
	search={props.searchNodes}
	resolve={props.resolveNode}
	findNode={props.findNode}
	suggest={props.copy}
/>
{#if nodeId !== ""}
	<SelectInput
		type={PointType.BindType}
		key={name}
		label="{name} point"
		options={[[PointType.Value, "value"]]}
		extra={target ? pointTypes(target) : []}
	/>
	<TextInput
		type={PointType.BindKey}
		key={name}
		label="{name} key"
		placeholder="0"
		suggestions={target ? pointKeys(target, type) : []}
	/>
{/if}
