import SerialDev from "#lib/icons/SerialDev.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.SerialDev,
	label: "Serial Device",
	icon: SerialDev,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "I",
	editor: Editor,
})
