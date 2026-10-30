<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText, getValue } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import FahrenheitInput from "./FahrenheitInput.svelte"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	const value = $derived(getValue(props.node.points, PointType.Value))
	const units = $derived(getText(props.node.points, PointType.Units))
	const valueText = $derived(String(Math.round(value * 100) / 100) + (units === "F" ? "°F" : "°C"))
</script>

<NodeEditor {props} icon={Io}>
	{#snippet header()}
		<span class="value">{valueText}</span>
	{/snippet}
	<Value label="ID">{getText(props.node.points, PointType.ID)}</Value>
	<TextInput type={PointType.Description} label="Description" />
	<FahrenheitInput label="Fahrenheit?" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<CounterWithReset
		type={PointType.ErrorCount}
		resetType={PointType.ErrorCountReset}
		label="Error Count"
	/>
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.value {
		padding: 0 7px;
	}
</style>
