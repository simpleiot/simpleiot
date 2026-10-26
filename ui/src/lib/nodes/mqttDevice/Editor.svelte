<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import PointTable from "#lib/components/PointTable.svelte"
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
	<Value label="Topic level">{identity}</Value>
	<TextInput type={PointType.Description} label="Description" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	{#if values.length > 0}
		<PointTable rows={values.map(formatValue)} label="Point" />
	{/if}
</NodeEditor>
