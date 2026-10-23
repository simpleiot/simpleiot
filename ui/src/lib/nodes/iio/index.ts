import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Iio,
	label: "IIO (analog IO)",
	icon: Io,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "G2",
})
