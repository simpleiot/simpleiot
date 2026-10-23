import CheckIcon from "@lucide/svelte/icons/check"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Condition,
	label: "Condition",
	icon: CheckIcon,
	parents: [NodeType.Rule],
	ownedBy: NodeType.Rule,
	sortKey: "A",
})
