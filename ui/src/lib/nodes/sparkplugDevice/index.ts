import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.SparkplugDevice,
	label: "Sparkplug Device",
	icon: Io,
	parents: [],
	ownedBy: NodeType.SparkplugNode,
	sortKey: "Z",
	editor: Editor,
})
