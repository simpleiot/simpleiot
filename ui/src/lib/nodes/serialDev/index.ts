import SerialDev from "#lib/icons/SerialDev.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.SerialDev,
	label: "Serial Device",
	icon: SerialDev,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "I",
})
