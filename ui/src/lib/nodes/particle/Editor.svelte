<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { sorted, type Point } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { toDateTimeString } from "#lib/util/iso8601.ts"
	import { filterSpecialPoints } from "#lib/siot/point.ts"
	import type { EditorProps } from "../types.ts"
	import Particle from "#lib/icons/Particle.svelte"

	let props: EditorProps = $props()

	interface MetricFormat {
		desc: (p: Point) => string
		vf: (p: Point) => string
	}

	const toOneDec = (p: Point) => p.value.toFixed(1)

	const metricFormatters: Record<string, MetricFormat | undefined> = {
		temp: { desc: () => "Temperature", vf: toOneDec },
	}

	function formatMetric(p: Point) {
		const f = metricFormatters[p.type]
		return {
			time: toDateTimeString(p.time),
			key: p.key,
			desc: f ? f.desc(p) : p.type,
			value: f ? f.vf(p) : toOneDec(p),
		}
	}

	const metrics = $derived(filterSpecialPoints(sorted(props.node.points)).map(formatMetric))
</script>

<NodeEditor {props} icon={Particle}>
	<div>Particle.io connection</div>
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.AuthToken} label="API Key" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<table class="data-table">
		<thead>
			<tr><th>Time</th><th>ID</th><th>Type</th><th class="num">Value</th></tr>
		</thead>
		<tbody>
			{#each metrics as m, i (i)}
				<tr><td>{m.time}</td><td>{m.key}</td><td>{m.desc}</td><td class="num">{m.value}</td></tr>
			{/each}
		</tbody>
	</table>
</NodeEditor>
