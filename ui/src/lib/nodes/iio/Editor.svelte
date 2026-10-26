<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getBool, getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	const disabled = $derived(getBool(props.node.points, PointType.Disabled))
	const connected = $derived(getBool(props.node.points, PointType.Connected))
	// the driver's own name for the device, which is what confirms the node
	// found what it was pointed at
	const deviceName = $derived(getText(props.node.points, PointType.DeviceName))
	const error = $derived(getText(props.node.points, PointType.Error))
</script>

{#snippet status(label: string, v: string)}
	<div class="status">{"  " + label + ": " + v}</div>
{/snippet}

<NodeEditor {props} icon={Io} labelWidth={180}>
	{#snippet header()}
		{#if deviceName !== ""}<span class="value">{deviceName}</span>{/if}
		{#if !disabled && !connected}<span>(not connected)</span>{/if}
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.Device} label="Device" placeholder="ads1015" />
	<NumberInput type={PointType.PollPeriod} label="Poll period (ms)" />
	<NumberInput type={PointType.SampleFrequency} label="Sample frequency (Hz)" />
	<NumberInput type={PointType.Oversampling} label="Oversampling ratio" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<NumberInput type={PointType.Debug} label="Debug level (0-9)" />
	<hr />
	{@render status("Connected", connected ? "yes" : "no")}
	{@render status("Device name", deviceName)}
	{@render status("Device path", getText(props.node.points, PointType.DevicePath))}
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
