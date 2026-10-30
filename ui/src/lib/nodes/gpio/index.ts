import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Gpio,
	label: "GPIO",
	icon: Io,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "G1",
	editor: Editor,
})
