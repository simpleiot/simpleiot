import CableIcon from "@lucide/svelte/icons/cable"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.NetworkManagerConn,
	label: "Connection",
	icon: CableIcon,
	parents: [NodeType.NetworkManager],
	ownedBy: NodeType.NetworkManager,
	sortKey: "E",
})
