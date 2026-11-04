<script lang="ts">
	// A link joins two widget ports on a canvas dashboard. It is drawn,
	// not run: a pipe between a pump and a tank describes the plant, and
	// implies nothing about data flow. Canvas drawing comes in a later
	// release; the points are editable here already.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import SplineIcon from "@lucide/svelte/icons/spline"

	let props: EditorProps = $props()
</script>

<NodeEditor {props} icon={SplineIcon}>
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.From} label="From" placeholder="widget ID.port" />
	<TextInput type={PointType.To} label="To" placeholder="widget ID.port" />
	<OptionInput
		type={PointType.Routing}
		label="Routing"
		options={[
			[PointValue.Direct, "direct"],
			[PointValue.Orthogonal, "orthogonal"],
		]}
	/>
	<KeyValueInput type={PointType.Vertex} label="Vertices" buttonLabel="Add vertex" />
</NodeEditor>
