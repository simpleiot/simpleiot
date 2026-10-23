<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
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
	import { filterSpecialPoints, renderPoint } from "#lib/siot/point.ts"
	import HardDriveIcon from "@lucide/svelte/icons/hard-drive"
	import Cloud from "@lucide/svelte/icons/cloud"
	import CloudOff from "@lucide/svelte/icons/cloud-off"
	import Power from "@lucide/svelte/icons/power"

	let props: EditorProps = $props()

	const sysState = $derived(getText(props.node.points, PointType.SysState))

	const latestPointTime = $derived(getLatest(props.node.points)?.time ?? epoch)

	function version(type: string, label: string): string {
		const p = get(props.node.points, type)
		return p ? label + ": " + p.text : ""
	}

	const versionHW = $derived(version(PointType.VersionHW, "HW"))
	const versionOS = $derived(version(PointType.VersionOS, "OS"))
	const versionApp = $derived(version(PointType.VersionApp, "App"))

	const shown = $derived(
		filterSpecialPoints(filterDeleted(sorted(props.node.points))).filter(
			(p) => p.type !== PointType.Tag
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
	<div class="points">
		{#each shown as p (p.type + ":" + p.key)}
			<div>{renderPoint(p)}</div>
		{/each}
	</div>
	<div>Last update: {toDateTimeString(latestPointTime)}</div>
	<div>
		Time since last update: {durationString(props.now - Date.parse(latestPointTime))}
	</div>
	{#if versionHW !== "" || versionOS !== "" || versionApp !== ""}
		<div>{"Version: " + versionHW + " " + versionOS + " " + versionApp}</div>
	{/if}
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<CheckboxInput type={PointType.StoreMetrics} label="Store metrics" />
	{#if getBool(props.node.points, PointType.StoreMetrics)}
		<NumberInput type={PointType.StoreMetricsPeriod} label="Metrics period (s)" />
	{/if}
</NodeEditor>

<style>
	.points {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 16px;
	}
</style>
