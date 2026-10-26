import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Iio,
	label: "IIO (analog IO)",
	icon: Io,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "G2",
	editor: Editor,
})
