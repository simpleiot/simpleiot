import NetworkIcon from "@lucide/svelte/icons/network"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.NetworkManager,
	label: "Network Manager",
	icon: NetworkIcon,
	parents: [NodeType.Device],
	hasChildren: true,
	sortKey: "R",
})
