<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import OnOffInput from "#lib/inputs/OnOffInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import {
		epoch,
		filterDeleted,
		getAll,
		getBool,
		getLatest,
		getText,
		sorted,
		type Point,
	} from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { toDateTimeString } from "#lib/util/iso8601.ts"
	import PointTable from "#lib/components/PointTable.svelte"
	import {
		descS,
		filterSpecialPoints,
		formatPoint,
		round,
		toWhole,
		type Format,
	} from "#lib/components/pointFormat.ts"
	import type { EditorProps } from "../types.ts"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	const points = $derived(props.node.points)
	const disabled = $derived(getBool(points, PointType.Disabled))
	const offline = $derived(getBool(points, PointType.Offline))
	const summary = $derived(
		`(${getText(points, PointType.Type)})  ${getText(points, PointType.Description)}`
	)

	const byKey = (pts: Point[]) =>
		pts.slice().sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0))

	const onOffArrays = $derived([
		["S:", byKey(getAll(points, PointType.Switch))],
		["L:", getAll(points, PointType.Light)],
		["I:", getAll(points, PointType.Input)],
	] as const)

	const deviceID = $derived(getText(points, PointType.DeviceID))
	const ip = $derived(getText(points, PointType.IP))
	const controlled = $derived(getBool(points, PointType.Controlled))
	const latestPointTime = $derived(getLatest(points)?.time ?? epoch)

	// a node is settable when it has points the client can set
	const isSettable = $derived(points.some((p) => p.type.includes("Set")))

	const controls = $derived(
		[PointType.Light, PointType.Switch].flatMap((t) =>
			byKey(points.filter((p) => p.type === `${t}Set`)).map((p) => ({
				key: p.key,
				type: t,
				setType: `${t}Set`,
			}))
		)
	)

	const onOffText = (p: Point) => (p.value === 0 ? "off" : "on")

	const formats: Record<string, Format> = {
		voltage: { desc: descS("Voltage"), vf: (p) => round(1, p.value) },
		temp: { desc: descS("Temperature (C)"), vf: (p) => round(1, p.value) },
		power: { desc: descS("Power"), vf: (p) => round(2, p.value) },
		current: { desc: descS("Current"), vf: (p) => round(2, p.value) },
		brightness: { desc: descS("Brightness"), vf: toWhole },
		lightTemp: { desc: descS("Light Temperature"), vf: toWhole },
		transition: { desc: descS("Transition"), vf: toWhole },
		white: { desc: descS("White"), vf: toWhole },
		energy: { desc: descS("Energy (Wh)"), vf: (p) => round(2, p.value) },
		humidity: { desc: descS("Humidity (%)"), vf: (p) => round(1, p.value) },
		position: { desc: descS("Position (%)"), vf: toWhole },
		coverState: { desc: descS("Cover State"), vf: (p) => p.text },
		battery: { desc: descS("Battery (V)"), vf: (p) => round(2, p.value) },
		batteryLevel: { desc: descS("Battery (%)"), vf: toWhole },
		externalPower: { desc: descS("External Power"), vf: onOffText },
		powerFactor: { desc: descS("Power Factor"), vf: (p) => round(2, p.value) },
		apparentPower: { desc: descS("Apparent Power (VA)"), vf: (p) => round(2, p.value) },
		frequency: { desc: descS("Frequency (Hz)"), vf: (p) => round(1, p.value) },
		alarm: { desc: descS("Alarm"), vf: onOffText },
	}

	const rows = $derived(
		filterSpecialPoints(filterDeleted(sorted(points))).map((p) => formatPoint(formats, p))
	)
</script>

<NodeEditor
	{props}
	icon={Io}
	description={summary}
	tone={disabled || offline ? "muted" : undefined}
>
	{#snippet header()}
		<span class="values">
			{#each onOffArrays as [label, pts] (label)}
				{#if pts.length > 0}
					<span>{label}</span>
					{#each pts as p (p.key)}
						<span class="onoff" class:on={p.value !== 0}>{onOffText(p)}</span>
					{/each}
				{/if}
			{/each}
		</span>
		{#if offline}<span>(offline)</span>{/if}
	{/snippet}
	<div class="display">ID: {deviceID}</div>
	<div class="display">
		IP: <a href="http://{ip}" target="_blank" rel="noopener noreferrer">{ip}</a>
	</div>
	<TextInput type={PointType.Description} label="Description" />
	{#if controlled}
		<div class="controls">
			{#each controls as c (`${c.type}.${c.key}`)}
				<OnOffInput key={c.key} type={c.type} setType={c.setType} label="{c.type} {c.key}" />
			{/each}
		</div>
	{/if}
	{#if isSettable}
		<CheckboxInput type={PointType.Controlled} label="Enable Control" />
	{/if}
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<div>Last update: {toDateTimeString(latestPointTime)}</div>
	{#if rows.length > 0}
		<PointTable {rows} label="Point" />
	{/if}
</NodeEditor>

<style>
	.values {
		display: flex;
	}

	.onoff {
		padding: 0 7px;
	}

	.onoff.on {
		background: var(--blue);
		color: var(--on-color);
	}

	.display {
		padding-left: 70px;
	}

	.display a {
		color: inherit;
		text-decoration: underline;
	}

	.controls {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
</style>
