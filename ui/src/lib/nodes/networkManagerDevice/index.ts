import RadioReceiverIcon from "@lucide/svelte/icons/radio-receiver"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.NetworkManagerDevice,
	label: "Network Device",
	icon: RadioReceiverIcon,
	parents: [],
	ownedBy: NodeType.NetworkManager,
	sortKey: "D",
})
