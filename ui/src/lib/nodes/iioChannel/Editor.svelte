<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText, getValue } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	const value = $derived(getValue(props.node.points, PointType.Value))
	const units = $derived(getText(props.node.points, PointType.Units))
	const valueText = $derived(
		String(Math.round(value * 1000) / 1000) + (units !== "" ? " " + units : "")
	)
	const error = $derived(getText(props.node.points, PointType.Error))
	const direction = $derived(getText(props.node.points, PointType.Direction))
	// only an output channel accepts a requested value
	const isOutput = $derived(direction === PointValue.Output)
</script>

{#snippet status(label: string, v: string)}
	<div class="status">{"  " + label + ": " + v}</div>
{/snippet}

<NodeEditor
	{props}
	icon={Io}
	labelWidth={180}
	description={getText(props.node.points, PointType.Description) + ": "}
>
	{#snippet header()}
		<span class="value">{valueText}</span>
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.Channel} label="Channel" placeholder="in_voltage0" />
	<NumberInput type={PointType.Scale} label="Scale" />
	<NumberInput type={PointType.Offset} label="Offset" />
	<TextInput type={PointType.Units} label="Units" />
	<NumberInput type={PointType.MinChange} label="Min change" />
	{#if isOutput}
		<NumberInput type={PointType.ValueSet} label="Value" />
	{/if}
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<hr />
	{@render status("Value", valueText)}
	{@render status("Channel type", getText(props.node.points, PointType.ChannelType))}
	{@render status("Direction", direction)}
	{#if error !== ""}{@render status("Error", error)}{/if}
	<CounterWithReset
		type={PointType.ErrorCount}
		resetType={PointType.ErrorCountReset}
		label="Error count"
	/>
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.value {
		padding: 0 7px;
	}

	hr {
		width: 100%;
		margin: 0;
		border: none;
		border-bottom: 2px solid var(--muted);
	}

	.status {
		white-space: pre-wrap;
	}
</style>
