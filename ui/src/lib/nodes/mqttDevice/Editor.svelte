<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { filterDeleted, getText, sorted, type Point } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { renderPoint2 } from "#lib/siot/point.ts"
	import type { EditorProps } from "../types.ts"
	import Io from "#lib/icons/Io.svelte"

	let props: EditorProps = $props()

	// Point types this editor renders on its own, above the value table.
	// Listing them again alongside the values the device publishes only makes
	// the values harder to find. The values themselves are value points,
	// which filterSpecialPoints would remove, so they are filtered here
	// instead.
	const configPoints: string[] = [PointType.Description, PointType.ID, PointType.Tag]

	const identity = $derived(getText(props.node.points, PointType.ID))
	const values = $derived(
		filterDeleted(sorted(props.node.points)).filter((p) => !configPoints.includes(p.type))
	)

	// A value point is named by its key, which the client builds from the
	// topic levels and payload fields below the ones the schema names. A
	// payload carrying a single measurement has no key, so the point type
	// names it.
	function formatValue(p: Point): { desc: string; value: string } {
		if (p.type === PointType.Value) {
			return {
				desc: p.key === "" ? p.type : p.key,
				value: p.text !== "" ? p.text : p.value.toFixed(2),
			}
		}
		return renderPoint2(p)
	}
</script>

<NodeEditor {props} icon={Io}>
	<div class="indent">Topic level: {identity}</div>
	<TextInput type={PointType.Description} label="Description" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	{#if values.length > 0}
		<table>
			<thead>
				<tr><th>Point</th><th>Value</th></tr>
			</thead>
			<tbody>
				{#each values.map(formatValue) as m, i (i)}
					<tr><td>{m.desc}</td><td class="num">{m.value}</td></tr>
				{/each}
			</tbody>
		</table>
	{/if}
</NodeEditor>

<style>
	.indent {
		padding-left: 70px;
	}

	table {
		margin: 7px;
		border-collapse: collapse;
	}

	th,
	td {
		padding: 5px 15px;
		border: 1px solid var(--fg);
	}

	th {
		font-weight: bold;
		text-align: center;
	}

	.num {
		text-align: right;
	}
</style>
