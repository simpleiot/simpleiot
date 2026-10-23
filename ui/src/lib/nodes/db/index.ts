import DatabaseIcon from "@lucide/svelte/icons/database"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Db,
	label: "Database",
	icon: DatabaseIcon,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "M",
})
