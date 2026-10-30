<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OnOffInput from "#lib/inputs/OnOffInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getBool, getText, getValue } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	const disabled = $derived(getBool(props.node.points, PointType.Disabled))
	const connected = $derived(getBool(props.node.points, PointType.Connected))
	const value = $derived(getBool(props.node.points, PointType.Value))
	// a simulated line has no hardware behind it, which is worth saying
	// where someone can see it without opening the node
	const isSim = $derived(getText(props.node.points, PointType.Chip) === PointValue.Sim)
	const valueText = $derived(value ? "on" : "off")
	// an unset direction is an input, matching the backend default
	const isOutput = $derived(getText(props.node.points, PointType.Direction) === PointValue.Output)
	const error = $derived(getText(props.node.points, PointType.Error))
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
		<span class="value" class:on={value}>{valueText}</span>
		{#if isSim}<span>(simulated)</span>{/if}
		{#if !disabled && !connected}<span>(not connected)</span>{/if}
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.Chip} label="Chip" placeholder="gpiochip0" />
	<TextInput type={PointType.Line} label="Line" placeholder="offset or name" />
	<OptionInput
		type={PointType.Direction}
		label="Direction"
		options={[
			[PointValue.Input, "Input"],
			[PointValue.Output, "Output"],
		]}
	/>
	{#if !isOutput}
		<OptionInput
			type={PointType.Bias}
			label="Bias"
			options={[
				["", "As is"],
				[PointValue.PullUp, "Pull up"],
				[PointValue.PullDown, "Pull down"],
				[PointValue.BiasDisabled, "Disabled"],
			]}
		/>
	{/if}
	{#if isOutput}
		<OptionInput
			type={PointType.Drive}
			label="Drive"
			options={[
				[PointValue.PushPull, "Push-pull"],
				[PointValue.OpenDrain, "Open drain"],
				[PointValue.OpenSource, "Open source"],
			]}
		/>
	{/if}
	<CheckboxInput type={PointType.ActiveLow} label="Active low" />
	{#if !isOutput}
		<NumberInput type={PointType.Debounce} label="Debounce (ms)" />
		<NumberInput type={PointType.PollPeriod} label="Poll period (ms)" />
	{/if}
	{#if isOutput}
		<CheckboxInput type={PointType.InitialValue} label="Initial value" />
		<OnOffInput type={PointType.Value} setType={PointType.ValueSet} label="Value" />
	{/if}
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<NumberInput type={PointType.Debug} label="Debug level (0-9)" />
	<hr />
	{@render status("Value", valueText)}
	{@render status("Connected", connected ? "yes" : "no")}
	{@render status(
		"Line offset",
		String(Math.round(getValue(props.node.points, PointType.LineOffset)))
	)}
	{@render status("Line name", getText(props.node.points, PointType.LineName))}
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

	.on {
		background: var(--blue);
		color: var(--on-color);
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
