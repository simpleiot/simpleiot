<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import RssIcon from "@lucide/svelte/icons/rss"

	let props: EditorProps = $props()

	const error = $derived(getText(props.node.points, PointType.Error))
</script>

<NodeEditor {props} icon={RssIcon} labelWidth={180}>
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.URI} label="Broker URI" placeholder="blank uses the built-in broker" />
	<TextInput
		type={PointType.TopicSchema}
		label="Topic schema"
		placeholder={"{site}/{gateway}/{device}"}
	/>
	<NumberInput type={PointType.MaxNodes} label="Max nodes (0 = 1000)" />
	<CheckboxInput type={PointType.Sparkplug} label="Sparkplug B" />
	<NumberInput type={PointType.Debug} label="Debug level (0-9)" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	{#if error !== ""}
		<div class="note">Error: {error}</div>
	{/if}
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.note {
		padding-left: 70px;
	}
</style>
