import BoxIcon from "@lucide/svelte/icons/box"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Widget,
	label: "Widget",
	icon: BoxIcon,
	parents: [NodeType.Dashboard],
	ownedBy: NodeType.Dashboard,
	sortKey: "A",
	editor: Editor,
})
