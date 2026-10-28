<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import ListInput from "#lib/inputs/ListInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText, sorted } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import PointTable from "#lib/components/PointTable.svelte"
	import {
		descKey,
		descS,
		filterSpecialPoints,
		formatPoint,
		round,
		toMiB,
		toPercent,
		toText,
		toTime,
		toWhole,
		type Format,
	} from "#lib/components/pointFormat.ts"
	import ChartNoAxesColumn from "@lucide/svelte/icons/chart-no-axes-column"

	let props: EditorProps = $props()

	const metricsType = $derived(getText(props.node.points, PointType.Type))

	// Point types this editor shows as configuration inputs above the
	// readings table. They are settings rather than readings, so listing
	// them alongside the readings only makes the readings harder to find.
	// This is the node's own configuration; filterSpecialPoints covers the
	// point types every node carries.
	const configPoints: readonly string[] = [
		PointType.Type,
		PointType.Name,
		PointType.Period,
		PointType.URI,
		PointType.Prefix,
		PointType.MaxSeries,
		PointType.CounterDelta,
		PointType.Tag,
	]

	const formats: Record<string, Format> = {
		metricAppAlloc: { desc: descS("App Memory Alloc"), vf: toMiB },
		metricAppNumGoroutine: { desc: descS("App Goroutine Count"), vf: toWhole },
		metricProcCPUPercent: { desc: descKey("Proc CPU %"), vf: toPercent },
		metricProcMemPercent: { desc: descKey("Proc Mem %"), vf: toPercent },
		metricProcMemRSS: { desc: descKey("Proc Mem RSS"), vf: toMiB },
		count: { desc: descKey("Proc Count"), vf: toWhole },
		host: { desc: descKey("Host"), vf: toText },
		hostBootTime: { desc: descS("Host Boot Time"), vf: toTime },
		metricSysCPUPercent: { desc: descS("Sys CPU %"), vf: toPercent },
		metricSysCPUFreq: { desc: descKey("CPU MHz"), vf: toWhole },
		metricSysCoolingState: { desc: descKey("Cooling State"), vf: toWhole },
		metricSysCoolingStateMax: { desc: descKey("Cooling State Max"), vf: toWhole },
		metricSysDiskUsedPercent: { desc: descKey("Disk Used %"), vf: toPercent },
		metricSysFanPWM: { desc: descKey("Fan PWM"), vf: toWhole },
		metricSysFanSpeed: { desc: descKey("Fan RPM"), vf: toWhole },
		metricSysLoad: { desc: descKey("Load"), vf: (p) => round(2, p.value) },
		metricSysMemUsedPercent: { desc: descS("Memory used %"), vf: toPercent },
		metricSysMem: { desc: descKey("Memory"), vf: toMiB },
		metricSysNetBytesRecv: { desc: descKey("Net RX"), vf: toWhole },
		metricSysNetBytesSent: { desc: descKey("Net TX"), vf: toWhole },
		metricSysUptime: { desc: descKey("Uptime"), vf: toWhole },
		temp: { desc: descKey("Temp"), vf: (p) => round(1, p.value) },
		voltage: { desc: descKey("Voltage"), vf: (p) => round(2, p.value) },
		current: { desc: descKey("Current"), vf: (p) => round(3, p.value) },
		power: { desc: descKey("Power"), vf: (p) => round(2, p.value) },
	}

	const rows = $derived(
		filterSpecialPoints(sorted(props.node.points))
			.filter((p) => !configPoints.includes(p.type))
			.map((p) => formatPoint(formats, p))
	)
</script>

<NodeEditor {props} icon={ChartNoAxesColumn} labelWidth={100}>
	<TextInput type={PointType.Description} label="Description" />
	<OptionInput
		type={PointType.Type}
		label="Type"
		options={[
			[PointValue.System, "system"],
			[PointValue.App, "this application"],
			[PointValue.Process, "named process"],
			[PointValue.Prometheus, "prometheus"],
			// modern systems have so many processes (1000's) and we need to
			// better be able to handle this point load for repeated points,
			// so all processes is left out for now
		]}
	/>
	{#if metricsType === PointValue.Process}
		<TextInput type={PointType.Name} label="proc name" />
	{/if}
	{#if metricsType === PointValue.Prometheus}
		<TextInput type={PointType.URI} label="URI" placeholder="http://127.0.0.1:9100/metrics" />
		<ListInput type={PointType.Prefix} label="Metric Prefixes" buttonLabel="Add Prefix" />
		<NumberInput type={PointType.MaxSeries} label="Max series (<= 3000)" />
		<CheckboxInput type={PointType.CounterDelta} label="Counter deltas" />
	{/if}
	<NumberInput type={PointType.Period} label="Period (s)" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<PointTable {rows} label="Metric" collapsible />
</NodeEditor>
