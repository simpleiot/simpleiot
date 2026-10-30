import Variable from "#lib/icons/Variable.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Variable,
	label: "Variable",
	icon: Variable,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "L",
	editor: Editor,
})
