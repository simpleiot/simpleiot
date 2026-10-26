import SendIcon from "@lucide/svelte/icons/send"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.MsgService,
	label: "Messaging Service",
	icon: SendIcon,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "J",
	editor: Editor,
})
