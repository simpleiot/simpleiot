<script lang="ts">
	// BindingInputs edits one binding of a widget: the node, through the
	// picker, then the point type and key, offered from what the chosen
	// node carries. When the chosen node sits below the dashboard's parent,
	// the binding can be made relative to it, so the same dashboard placed
	// under another parent follows that parent's nodes.
	import NodeRefInput from "#lib/inputs/NodeRefInput.svelte"
	import SelectInput from "#lib/inputs/SelectInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { inputContext, editTime } from "#lib/inputs/context.ts"
	import { pointKeys, pointTypes, type Node } from "#lib/siot/node.ts"
	import { getText, newText, newTombstone } from "#lib/siot/point.ts"
	import { NodeType, PointType } from "#lib/siot/schema.ts"
	import type { NodeMatch } from "#lib/siot/search.ts"
	import { pathBelow } from "../dashboard/paths.ts"
	import type { EditorProps } from "../types.ts"

	let {
		name,
		props,
		via,
		source = false,
		dashboard = false,
		pointType = PointType.Value,
	}: {
		name: string
		props: EditorProps
		// via is the dashboard's parent, which a relative path starts from
		via?: { id: string; description: string }
		// source says the binding holds a document or an image the widget
		// draws, which a file node carries
		source?: boolean
		// dashboard says the binding names a dashboard, to show in place or
		// to open
		dashboard?: boolean
		// pointType is the point the binding reads when none is chosen
		pointType?: string
	} = $props()

	const ctx = inputContext()

	// a binding reads a node's points, which a dashboard, widget, or link
	// has none of; a binding that names a dashboard wants dashboards alone,
	// and a source binding file nodes alone
	const layoutTypes: readonly string[] = [NodeType.Dashboard, NodeType.Widget, NodeType.Link]
	const search = (query: string) =>
		dashboard
			? props.searchNodes(query, NodeType.Dashboard)
			: source
				? props.searchNodes(query, NodeType.File)
				: props.searchNodes(query).then((m) => m.filter((x) => !layoutTypes.includes(x.type)))
	const points = $derived(props.node.points)
	const nodeId = $derived(getText(points, PointType.BindNode, name))
	const path = $derived(getText(points, PointType.BindPath, name))
	const type = $derived(getText(points, PointType.BindType, name) || pointType)

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

	// the path the last pick would have below the via, when there is one
	let picked = $state<NodeMatch | null>(null)
	const relative = $derived(
		via && picked?.id === nodeId ? pathBelow(picked.chain, via.id) : undefined
	)

	function setPath(p: string) {
		ctx.onEditPoints([
			p === ""
				? newTombstone(PointType.BindPath, name, editTime(ctx))
				: newText(PointType.BindPath, name, p, editTime(ctx)),
		])
	}
</script>

<NodeRefInput
	type={PointType.BindNode}
	key={name}
	label="{name} node"
	{search}
	resolve={props.resolveNode}
	findNode={props.findNode}
	suggest={props.copy}
	onPick={(m: NodeMatch) => (picked = m)}
/>
{#if path !== ""}
	<div class="relative">
		<span class="path" title="relative to {via?.description ?? "the dashboard's parent"}"
			>relative: {path}</span
		>
		<button class="btn gray" type="button" onclick={() => setPath("")}>absolute</button>
	</div>
{:else if relative !== undefined && via}
	<div class="relative">
		<button class="btn gray" type="button" onclick={() => setPath(relative)}>
			make relative to {via.description}
		</button>
		<span class="path">{relative}</span>
	</div>
{/if}
<!-- a source binding reads the document and nothing else -->
{#if !source && (nodeId !== "" || path !== "")}
	<SelectInput
		type={PointType.BindType}
		key={name}
		label="{name} point"
		options={[[pointType, pointType]]}
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

<style>
	.relative {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
		padding-left: 160px;
	}

	.path {
		font-family: var(--mono);
		font-size: 13px;
		color: var(--muted);
	}

	@media (max-width: 600px) {
		.relative {
			padding-left: 0;
		}
	}
</style>
