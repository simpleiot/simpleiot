<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import HashIcon from "@lucide/svelte/icons/hash"

	let props: EditorProps = $props()

	const topic = $derived(getText(props.node.points, PointType.Topic))
	const error = $derived(getText(props.node.points, PointType.Error))
</script>

<NodeEditor {props} icon={HashIcon} labelWidth={180}>
	{#snippet header()}
		<span class="topic">{topic}</span>
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.Topic} label="Topic" placeholder="plant/line3/tank" />
	<TextInput
		type={PointType.Path}
		label="JSON path"
		placeholder="$.value (blank maps the whole payload)"
	/>
	<TextInput type={PointType.Units} label="Units" />
	<NumberInput type={PointType.Scale} label="Scale" />
	<NumberInput type={PointType.Offset} label="Offset" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	{#if error !== ""}
		<div class="note">Error: {error}</div>
	{/if}
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.topic {
		padding: 0 7px;
	}

	.note {
		padding-left: 70px;
	}
</style>
