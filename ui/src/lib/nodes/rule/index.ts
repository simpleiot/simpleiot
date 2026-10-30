import ListIcon from "@lucide/svelte/icons/list"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Rule,
	label: "Rule",
	icon: ListIcon,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "E",
	editor: Editor,
})
