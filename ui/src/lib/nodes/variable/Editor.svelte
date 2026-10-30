<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OnOffInput from "#lib/inputs/OnOffInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText, getValue } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Variable from "#lib/icons/Variable.svelte"

	let props: EditorProps = $props()

	const value = $derived(getValue(props.node.points, PointType.Value))
	const variableType = $derived(getText(props.node.points, PointType.VariableType))
	const valueText = $derived.by(() => {
		if (variableType === PointValue.Number) {
			return String(Math.round(value * 100) / 100)
		}
		if (variableType === PointValue.Text) {
			return getText(props.node.points, PointType.Value)
		}
		return value === 0 ? "off" : "on"
	})
</script>

<NodeEditor {props} icon={Variable}>
	{#snippet header()}
		<span class="value" class:on={valueText === "on"}
			>{valueText +
				(variableType === PointValue.Number
					? " " + getText(props.node.points, PointType.Units)
					: "")}</span
		>
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<OptionInput
		type={PointType.VariableType}
		label="Variable type"
		options={[
			[PointValue.OnOff, "On/Off"],
			[PointValue.Number, "Number"],
			[PointValue.Text, "Text"],
		]}
	/>
	{#if variableType === PointValue.OnOff}
		<OnOffInput type={PointType.Value} setType={PointType.Value} label="Value" />
	{/if}
	{#if variableType === PointValue.Number}
		<NumberInput type={PointType.Value} label="Value" />
		<TextInput type={PointType.Units} label="Units" />
	{/if}
	{#if variableType === PointValue.Text}
		<TextInput type={PointType.Value} label="Text" />
	{/if}
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
</style>
