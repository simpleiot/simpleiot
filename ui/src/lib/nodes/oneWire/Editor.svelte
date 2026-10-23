<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getValue } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import OneWire from "#lib/icons/OneWire.svelte"

	let props: EditorProps = $props()

	const index = $derived(getValue(props.node.points, PointType.Index))
</script>

<NodeEditor {props} icon={OneWire} labelWidth={180}>
	<Value label="Bus #">{index}</Value>
	<TextInput type={PointType.Description} label="Description" />
	<NumberInput type={PointType.Debug} label="Debug level (0-9)" />
	<NumberInput type={PointType.PollPeriod} label="Poll period (ms)" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<CounterWithReset
		type={PointType.ErrorCount}
		resetType={PointType.ErrorCountReset}
		label="Error Count"
	/>
</NodeEditor>
