import Bus from "#lib/icons/Bus.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Modbus,
	label: "Modbus",
	icon: Bus,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "D",
})
