<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import ButtonAction from "#lib/inputs/ButtonAction.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getBool, getText, getValue } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import MapPin from "@lucide/svelte/icons/map-pin"

	let props: EditorProps = $props()

	const labelWidth = 180

	// Fix type is stored as a number so it can be graphed and stored in
	// metrics-only databases. The encoding follows gpsd's TPV mode field.
	function fixTypeLabel(v: number): string {
		switch (Math.round(v)) {
			case 2:
				return "2D"
			case 3:
				return "3D"
			default:
				return "no fix"
		}
	}

	// Fix quality is likewise numeric, following the NMEA GGA fix quality
	// encoding, which covers everything the three sources report.
	const fixQualities: Record<number, string> = {
		1: "GPS",
		2: "DGPS",
		3: "PPS",
		4: "RTK fixed",
		5: "RTK float",
		6: "estimated",
		7: "manual",
		8: "simulated",
	}
	const fixQualityLabel = (v: number) => fixQualities[Math.round(v)] ?? "none"

	// Coordinates are shown to five decimal places, which is roughly a meter.
	const coordinate = (v: number) => v.toFixed(5)

	const points = $derived(props.node.points)
	const disabled = $derived(getBool(points, PointType.Disabled))
	const connected = $derived(getBool(points, PointType.Connected))
	const latitude = $derived(getValue(points, PointType.Latitude))
	const longitude = $derived(getValue(points, PointType.Longitude))

	const source = $derived(getText(points, PointType.GpsSource))
	// an unset source behaves as serial, matching the backend default
	const isSerial = $derived(source === PointValue.GpsSourceSerial || source === "")
	const isGpsd = $derived(source === PointValue.GpsSourceGpsd)
	const isSim = $derived(source === PointValue.GpsSourceSim)

	const rounded = (places: number, type: string) => getValue(points, type).toFixed(places)

	const status = $derived([
		["Latitude", coordinate(latitude)],
		["Longitude", coordinate(longitude)],
		["Altitude (m)", rounded(1, PointType.Altitude)],
		["Speed (m/s)", rounded(2, PointType.Speed)],
		["Heading (deg)", rounded(1, PointType.Heading)],
		["Fix type", fixTypeLabel(getValue(points, PointType.FixType))],
		["Fix quality", fixQualityLabel(getValue(points, PointType.FixQuality))],
		["Satellites", rounded(0, PointType.NumSat)],
		["HDOP", rounded(2, PointType.Hdop)],
		["Connected", connected ? "yes" : "no"],
	])
</script>

<NodeEditor {props} icon={MapPin} {labelWidth} tone={disabled || !connected ? "muted" : undefined}>
	{#snippet header()}
		<span class="coords">{coordinate(latitude)}, {coordinate(longitude)}</span>
		{#if !disabled && !connected}<span>(not connected)</span>{/if}
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<OptionInput
		type={PointType.GpsSource}
		label="Source"
		options={[
			[PointValue.GpsSourceSerial, "Serial (NMEA)"],
			[PointValue.GpsSourceGpsd, "gpsd"],
			[PointValue.GpsSourceSim, "Simulated"],
		]}
	/>
	{#if isSerial}
		<TextInput type={PointType.Port} label="Port" placeholder="/dev/ttyUSB0" />
		<TextInput type={PointType.Baud} label="Baud" placeholder="9600" />
	{/if}
	{#if isGpsd}
		<TextInput type={PointType.GpsdAddress} label="gpsd address" placeholder="localhost:2947" />
		<TextInput type={PointType.Device} label="Device" placeholder="(blank watches all)" />
	{/if}
	{#if isSim}
		<NumberInput type={PointType.SimLatitude} label="Start latitude" />
		<NumberInput type={PointType.SimLongitude} label="Start longitude" />
		<NumberInput type={PointType.SimSpeed} label="Speed (m/s)" />
		<NumberInput type={PointType.SimHeading} label="Start heading (deg)" />
		<NumberInput type={PointType.SimHeadingRate} label="Heading drift (deg/s)" />
		<NumberInput type={PointType.Period} label="Update period (s)" />
		<div class="button-row-gps">
			<span style:width="{labelWidth}px"></span>
			<ButtonAction type={PointType.SimReset} value={1} label="Reset location" />
		</div>
	{/if}
	<NumberInput type={PointType.Debug} label="Debug level (0-9)" />
	<div class="rule"></div>
	{#each status as [label, value] (label)}
		<div class="status">{`  ${label}: ${value}`}</div>
	{/each}
	<CounterWithReset type={PointType.Rx} resetType={PointType.RxReset} label="Rx count" />
	<CounterWithReset
		type={PointType.ErrorCount}
		resetType={PointType.ErrorCountReset}
		label="Error count"
	/>
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.coords {
		padding: 0 7px;
	}

	.button-row-gps {
		display: flex;
		gap: 20px;
	}

	.rule {
		border-bottom: 2px solid var(--muted);
	}

	.status {
		white-space: pre;
	}
</style>
