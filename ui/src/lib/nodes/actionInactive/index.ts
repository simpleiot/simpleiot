import TrendingDownIcon from "@lucide/svelte/icons/trending-down"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.ActionInactive,
	label: "Action (rule inactive)",
	icon: TrendingDownIcon,
	parents: [NodeType.Rule],
	ownedBy: NodeType.Rule,
	sortKey: "C",
})
