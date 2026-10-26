<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import ListInput from "#lib/inputs/ListInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import DatabaseIcon from "@lucide/svelte/icons/database"

	let props: EditorProps = $props()

	const victoriaMetrics = $derived(
		getText(props.node.points, PointType.DbType) === PointValue.VictoriaMetrics
	)
	const urlPlaceholder = $derived(
		victoriaMetrics ? "http://myserver:8428" : "https://myserver:8086"
	)
</script>

<NodeEditor {props} icon={DatabaseIcon}>
	<OptionInput
		type={PointType.DbType}
		label="Database Type"
		options={[
			[PointValue.InfluxDb, "InfluxDB 2.x"],
			[PointValue.VictoriaMetrics, "Victoria Metrics"],
		]}
	/>
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.URI} label="URL" placeholder={urlPlaceholder} />
	{#if !victoriaMetrics}
		<TextInput type={PointType.Org} label="Organization" placeholder="org name" />
		<TextInput type={PointType.Bucket} label="Bucket" placeholder="bucket name" />
	{/if}
	<TextInput type={PointType.AuthToken} label="Auth Token" />
	<ListInput type={PointType.TagPointType} label="Tag Point Types" buttonLabel="Add Point Type" />
	<CheckboxInput type={PointType.ExpandKeyLabels} label="Expand Key Labels" />
</NodeEditor>
