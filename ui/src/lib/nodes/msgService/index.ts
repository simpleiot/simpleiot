import SendIcon from "@lucide/svelte/icons/send"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.MsgService,
	label: "Messaging Service",
	icon: SendIcon,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "J",
})
