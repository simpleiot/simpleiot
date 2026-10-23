import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Gpio,
	label: "GPIO",
	icon: Io,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "G1",
})
