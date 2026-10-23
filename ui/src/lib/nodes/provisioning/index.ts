import ListIcon from "@lucide/svelte/icons/list"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Provisioning,
	label: "Provisioning",
	icon: ListIcon,
	parents: [],
	hasChildren: true,
	editor: Editor,
})
