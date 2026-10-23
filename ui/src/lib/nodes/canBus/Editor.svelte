<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getValue } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import SerialDev from "#lib/icons/SerialDev.svelte"

	let props: EditorProps = $props()

	const labelWidth = 180

	const round2 = (v: number) => Math.round(v * 100) / 100
	const msgsInDb = $derived(round2(getValue(props.node.points, PointType.MsgsInDb)))
	const signalsInDb = $derived(round2(getValue(props.node.points, PointType.SignalsInDb)))
</script>

<NodeEditor {props} icon={SerialDev} {labelWidth}>
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.Device} label="Device" placeholder="can0" />
	<TextInput type={PointType.BitRate} label="Bit rate" placeholder="250000" />
	<Value label="Messages in db">{msgsInDb}</Value>
	<Value label="Signals in db">{signalsInDb}</Value>
	<CounterWithReset
		type={PointType.MsgsRecvdDb}
		resetType={PointType.MsgsRecvdDbReset}
		label="Db msgs received"
	/>
	<CounterWithReset
		type={PointType.MsgsRecvdOther}
		resetType={PointType.MsgsRecvdOtherReset}
		label="Other msgs recvd"
	/>
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>
