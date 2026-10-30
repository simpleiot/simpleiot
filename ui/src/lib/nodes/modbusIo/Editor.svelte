<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OnOffInput from "#lib/inputs/OnOffInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText, getValue } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	const points = $derived(props.node.points)
	const modbusIOType = $derived(getText(points, PointType.ModbusIOType))
	const isClient = $derived(
		props.parent !== undefined &&
			getText(props.parent.points, PointType.ClientServer) === PointValue.Client
	)
	const isWrite = $derived(
		modbusIOType === PointValue.ModbusHoldingRegister || modbusIOType === PointValue.ModbusCoil
	)
	const value = $derived(getValue(points, PointType.Value))
	const valueSet = $derived(getValue(points, PointType.ValueSet))
	const isRegister = $derived(
		modbusIOType === PointValue.ModbusInputRegister ||
			modbusIOType === PointValue.ModbusHoldingRegister
	)
	const isReadOnly = $derived(getValue(points, PointType.ReadOnly) === 1)
	const valueText = $derived(
		isRegister ? String(Math.round(value * 100) / 100) : value === 0 ? "off" : "on"
	)
</script>

<NodeEditor {props} icon={Io} description="{getText(points, PointType.Description)}:">
	{#snippet header()}
		<span class="value" class:on={valueText === "on"}
			>{valueText}{isRegister ? ` ${getText(points, PointType.Units)}` : ""}</span
		>
		{#if isClient && isWrite && !isReadOnly && value !== valueSet}
			<span>(cmd pending)</span>
		{/if}
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	{#if isClient}
		<NumberInput type={PointType.ID} label="ID" />
	{/if}
	<NumberInput type={PointType.Address} label="Address" />
	<OptionInput
		type={PointType.ModbusIOType}
		label="IO type"
		options={[
			[PointValue.ModbusDiscreteInput, "discrete input (r)"],
			[PointValue.ModbusCoil, "coil (rw)"],
			[PointValue.ModbusInputRegister, "input register(r)"],
			[PointValue.ModbusHoldingRegister, "holding register(rw)"],
		]}
	/>
	{#if isClient && isWrite}
		<CheckboxInput type={PointType.ReadOnly} label="Read only" />
	{/if}
	{#if isRegister}
		<NumberInput type={PointType.Scale} label="Scale factor" />
		<NumberInput type={PointType.Offset} label="Offset" />
		<TextInput type={PointType.Units} label="Units" />
		<OptionInput
			type={PointType.DataFormat}
			label="Data format"
			options={[
				[PointValue.UINT16, "UINT16"],
				[PointValue.INT16, "INT16"],
				[PointValue.UINT32, "UINT32"],
				[PointValue.INT32, "INT32"],
				[PointValue.FLOAT32, "FLOAT32"],
			]}
		/>
	{/if}
	<!--
		The client sets coils and holding registers, and the server (device)
		sets discrete inputs and input registers. Some devices also have read
		only coils and holding registers; the client and server cannot both
		set a value.
	-->
	{#if isClient && modbusIOType === PointValue.ModbusHoldingRegister && !isReadOnly}
		<NumberInput type={PointType.ValueSet} label="Value" />
	{/if}
	{#if isClient && modbusIOType === PointValue.ModbusCoil && !isReadOnly}
		<OnOffInput type={PointType.Value} setType={PointType.ValueSet} label="Value" />
	{/if}
	{#if !isClient && modbusIOType === PointValue.ModbusInputRegister}
		<NumberInput type={PointType.Value} label="Value" />
	{/if}
	{#if !isClient && modbusIOType === PointValue.ModbusDiscreteInput}
		<OnOffInput type={PointType.Value} setType={PointType.Value} label="Value" />
	{/if}
	{#if isClient}
		<CheckboxInput type={PointType.Disabled} label="Disabled" />
	{/if}
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

<style>
	.value {
		padding: 0 7px;
	}

	.value.on {
		background: var(--blue);
		color: var(--on-color);
	}
</style>
