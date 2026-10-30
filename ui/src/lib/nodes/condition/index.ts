import CheckIcon from "@lucide/svelte/icons/check"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Condition,
	label: "Condition",
	icon: CheckIcon,
	parents: [NodeType.Rule],
	ownedBy: NodeType.Rule,
	sortKey: "A",
	editor: Editor,
})
