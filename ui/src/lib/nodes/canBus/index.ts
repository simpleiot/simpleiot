import SerialDev from "#lib/icons/SerialDev.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.CanBus,
	label: "CAN Bus",
	icon: SerialDev,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "H",
	editor: Editor,
})
