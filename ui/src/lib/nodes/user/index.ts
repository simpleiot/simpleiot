import UserIcon from "@lucide/svelte/icons/user"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.User,
	label: "User",
	icon: UserIcon,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "B",
})
