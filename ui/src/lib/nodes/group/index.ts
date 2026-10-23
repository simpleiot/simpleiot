import UsersIcon from "@lucide/svelte/icons/users"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Group,
	label: "Group",
	icon: UsersIcon,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "C",
})
