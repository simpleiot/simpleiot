<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { get, getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import CloudIcon from "@lucide/svelte/icons/cloud"
	import CloudOffIcon from "@lucide/svelte/icons/cloud-off"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	const online = $derived(get(props.node.points, PointType.Online))
	const identity = $derived(getText(props.node.points, PointType.ID))
</script>

<NodeEditor {props} icon={Io}>
	{#snippet lead()}
		{#if online?.value === 0}
			<CloudOffIcon size={20} />
		{:else if online?.value === 1}
			<CloudIcon size={20} />
		{/if}
	{/snippet}
	<Value label="Sparkplug device">{identity}</Value>
	<TextInput type={PointType.Description} label="Description" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>
