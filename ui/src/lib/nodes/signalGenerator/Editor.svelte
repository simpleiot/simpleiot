<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getBool, getText, getValue } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import ActivityIcon from "@lucide/svelte/icons/activity"

	let props: EditorProps = $props()

	// signal types
	const valueSine = "sine"
	const valueSquare = "square"
	const valueTriangle = "triangle"
	const valueRandomWalk = "random walk"

	// keys of the destination point
	const keyParent = "parent"
	const keyHighRate = "highRate"
	const keyPointType = "pointType"
	const keyPointKey = "pointKey"

	// the generated points land on the destination point type/key, which
	// defaults to "value" when not configured (see client/signal-generator.go)
	const destPointType = $derived(
		getText(props.node.points, PointType.Destination, keyPointType) || PointType.Value
	)
	const destPointKey = $derived(getText(props.node.points, PointType.Destination, keyPointKey))
	const value = $derived(getValue(props.node.points, destPointType, destPointKey))
	const valueText = $derived(String(Math.round(value * 100) / 100))
	const disabled = $derived(getBool(props.node.points, PointType.Disabled))
	const signalType = $derived(getText(props.node.points, PointType.SignalType))
</script>

<NodeEditor {props} icon={ActivityIcon} tone={disabled ? "muted" : undefined}>
	{#snippet header()}
		<span class="value">{valueText + " " + getText(props.node.points, PointType.Units)}</span>
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<TextInput type={PointType.Units} label="Units" />
	<OptionInput
		type={PointType.SignalType}
		label="Signal type"
		options={[
			[valueSine, "Sine"],
			[valueSquare, "Square"],
			[valueTriangle, "Triangle"],
			[valueRandomWalk, "Random Walk"],
		]}
	/>
	<NumberInput type={PointType.MinValue} label="Min. Value" />
	<NumberInput type={PointType.MaxValue} label="Max. Value" />
	<NumberInput type={PointType.InitialValue} label="Initial Value" />
	<NumberInput type={PointType.RoundTo} label="Round To" />
	<NumberInput type={PointType.SampleRate} label="Sample Rate (Hz)" />
	<CheckboxInput type={PointType.Destination} key={keyParent} label="Sync parent node" />
	<CheckboxInput type={PointType.Destination} key={keyHighRate} label="High rate data" />
	<TextInput type={PointType.Destination} key={keyPointType} label="Point type" />
	<TextInput type={PointType.Destination} key={keyPointKey} label="Point key" />
	<NumberInput type={PointType.BatchPeriod} label="Batch Period (ms)" />
	{#if signalType === valueSine || signalType === valueSquare || signalType === valueTriangle}
		<NumberInput type={PointType.Frequency} label="Frequency (Hz)" />
	{/if}
	{#if signalType === valueRandomWalk}
		<NumberInput type={PointType.MinIncrement} label="Min. Increment" />
		<NumberInput type={PointType.MaxIncrement} label="Max. Increment" />
	{/if}
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.value {
		padding: 0 7px;
	}
</style>
