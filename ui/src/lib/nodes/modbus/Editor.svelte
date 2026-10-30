<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Bus from "#lib/icons/Bus.svelte"

	let props: EditorProps = $props()

	const clientServer = $derived(getText(props.node.points, PointType.ClientServer))
	const protocol = $derived(getText(props.node.points, PointType.Protocol))
</script>

<NodeEditor {props} icon={Bus} labelWidth={180}>
	<TextInput type={PointType.Description} label="Description" />
	<OptionInput
		type={PointType.ClientServer}
		label="Client/Server"
		options={[
			[PointValue.Client, "client"],
			[PointValue.Server, "server"],
		]}
	/>
	<OptionInput
		type={PointType.Protocol}
		label="Protocol"
		options={[
			[PointValue.RTU, "RTU"],
			[PointValue.TCP, "TCP"],
		]}
	/>
	{#if protocol === PointValue.RTU}
		<TextInput type={PointType.Port} label="Port" placeholder="/dev/ttyUSB0" />
	{/if}
	{#if protocol === PointValue.TCP && clientServer === PointValue.Server}
		<TextInput type={PointType.Port} label="Port" placeholder="502" />
	{/if}
	{#if protocol === PointValue.TCP && clientServer === PointValue.Client}
		<TextInput type={PointType.URI} label="URI" placeholder="192.168.1.201:502" />
	{/if}
	{#if protocol === PointValue.RTU}
		<TextInput type={PointType.Baud} label="Baud" placeholder="9600" />
	{/if}
	{#if clientServer === PointValue.Server}
		<NumberInput type={PointType.ID} label="Device ID" />
	{/if}
	{#if clientServer === PointValue.Client}
		<NumberInput type={PointType.PollPeriod} label="Poll period (ms)" />
	{/if}
	<NumberInput type={PointType.Timeout} label="Resp timeout (ms)" />
	<NumberInput type={PointType.Debug} label="Debug level (0-9)" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<CounterWithReset
		type={PointType.ErrorCount}
		resetType={PointType.ErrorCountReset}
		label="Error Count"
	/>
	<CounterWithReset
		type={PointType.ErrorCountEOF}
		resetType={PointType.ErrorCountEOFReset}
		label="EOF Error Count"
	/>
	<CounterWithReset
		type={PointType.ErrorCountCRC}
		resetType={PointType.ErrorCountCRCReset}
		label="CRC Error Count"
	/>
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>
