import TrendingUpIcon from "@lucide/svelte/icons/trending-up"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Action,
	label: "Action (rule active)",
	icon: TrendingUpIcon,
	parents: [NodeType.Rule],
	ownedBy: NodeType.Rule,
	sortKey: "B",
	editor: Editor,
})
