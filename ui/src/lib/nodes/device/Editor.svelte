<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import { description } from "#lib/siot/node.ts"
	import {
		epoch,
		filterDeleted,
		get,
		getBool,
		getLatest,
		getText,
		newText,
		sorted,
	} from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { durationString } from "#lib/util/duration.ts"
	import { toDateTimeString } from "#lib/util/iso8601.ts"
	import type { EditorProps } from "../types.ts"
	import { filterSpecialPoints, renderPoint2 } from "#lib/siot/point.ts"
	import PointTable from "#lib/components/PointTable.svelte"
	import HardDriveIcon from "@lucide/svelte/icons/hard-drive"
	import Cloud from "@lucide/svelte/icons/cloud"
	import CloudOff from "@lucide/svelte/icons/cloud-off"
	import Power from "@lucide/svelte/icons/power"

	let props: EditorProps = $props()

	const sysState = $derived(getText(props.node.points, PointType.SysState))
	// an upstream sets sysState when the device connects or drops, so the
	// point's time is when it came online or went offline
	const sysStateTime = $derived(get(props.node.points, PointType.SysState)?.time ?? epoch)
	const status = $derived(
		(sysState === "online" ? "Online" : "Offline") +
			` since ${toDateTimeString(sysStateTime)}` +
			` (${durationString(props.now - Date.parse(sysStateTime))} ago)`
	)

	const latestPointTime = $derived(getLatest(props.node.points)?.time ?? epoch)

	function version(type: string, label: string): string {
		const p = get(props.node.points, type)
		return p ? label + ": " + p.text : ""
	}

	const versionHW = $derived(version(PointType.VersionHW, "HW"))
	const versionOS = $derived(version(PointType.VersionOS, "OS"))
	const versionApp = $derived(version(PointType.VersionApp, "App"))

	const storeMetrics = $derived(getBool(props.node.points, PointType.StoreMetrics))

	// metricNats* points left over from before store metrics were optional
	// are not updated while store metrics are off, so they are hidden
	const shown = $derived(
		filterSpecialPoints(filterDeleted(sorted(props.node.points))).filter(
			(p) =>
				p.type !== PointType.Tag &&
				p.type !== PointType.SysState &&
				(storeMetrics || !p.type.startsWith("metricNats"))
		)
	)

	function oninput(e: Event & { currentTarget: HTMLInputElement }) {
		props.onEditPoints([
			newText(PointType.Description, "0", e.currentTarget.value, new Date(props.now).toISOString()),
		])
	}
</script>

<NodeEditor {props} icon={HardDriveIcon} description="">
	{#snippet header()}
		<!-- sysState values are not in the schema -->
		{#if sysState === "powerOff"}
			<Power size={20} />
		{:else if sysState === "offline"}
			<CloudOff size={20} />
		{:else if sysState === "online"}
			<Cloud size={20} />
		{/if}
		<input
			type="text"
			value={description(props.node)}
			placeholder="node description"
			aria-label="node description"
			{oninput}
		/>
	{/snippet}
	{#if sysState === "online" || sysState === "offline"}
		<Value label="Status">{status}</Value>
	{/if}
	{#if shown.length > 0}
		<PointTable rows={shown.map(renderPoint2)} label="Point" />
	{/if}
	<!-- the store metrics are what keep this node's points current -->
	{#if storeMetrics}
		<Value label="Last update">{toDateTimeString(latestPointTime)}</Value>
		<Value label="Since update">{durationString(props.now - Date.parse(latestPointTime))}</Value>
	{/if}
	{#if versionHW !== "" || versionOS !== "" || versionApp !== ""}
		<Value label="Version">{[versionHW, versionOS, versionApp].join(" ").trim()}</Value>
	{/if}
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<CheckboxInput type={PointType.StoreMetrics} label="Store metrics" />
	{#if storeMetrics}
		<NumberInput type={PointType.StoreMetricsPeriod} label="Metrics period (s)" />
	{/if}
</NodeEditor>
