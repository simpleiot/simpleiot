import RadioReceiverIcon from "@lucide/svelte/icons/radio-receiver"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.NetworkManagerDevice,
	label: "Network Device",
	icon: RadioReceiverIcon,
	parents: [],
	ownedBy: NodeType.NetworkManager,
	sortKey: "D",
	editor: Editor,
})
