import ListIcon from "@lucide/svelte/icons/list"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Provisioning,
	label: "Provisioning",
	icon: ListIcon,
	parents: [],
	hasChildren: true,
})
