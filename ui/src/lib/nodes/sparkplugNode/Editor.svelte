<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import CloudIcon from "@lucide/svelte/icons/cloud"
	import CloudOffIcon from "@lucide/svelte/icons/cloud-off"
	import RadioReceiverIcon from "@lucide/svelte/icons/radio-receiver"

	let props: EditorProps = $props()

	const sysState = $derived(getText(props.node.points, PointType.SysState))
	const identity = $derived(getText(props.node.points, PointType.ID))
</script>

<NodeEditor {props} icon={RadioReceiverIcon}>
	{#snippet lead()}
		{#if sysState === "offline"}
			<CloudOffIcon size={20} />
		{:else if sysState === "online"}
			<CloudIcon size={20} />
		{/if}
	{/snippet}
	<div class="note">Sparkplug edge node: {identity}</div>
	<TextInput type={PointType.Description} label="Description" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.note {
		padding-left: 70px;
	}
</style>
