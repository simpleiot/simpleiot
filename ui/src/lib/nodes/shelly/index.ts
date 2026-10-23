import Shelly from "#lib/icons/Shelly.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Shelly,
	label: "Shelly",
	icon: Shelly,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "P",
	editor: Editor,
})
