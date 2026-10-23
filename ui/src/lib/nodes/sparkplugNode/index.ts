import RadioReceiverIcon from "@lucide/svelte/icons/radio-receiver"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.SparkplugNode,
	label: "Sparkplug Node",
	icon: RadioReceiverIcon,
	parents: [],
	ownedBy: NodeType.SparkplugGroup,
	sortKey: "Y",
	editor: Editor,
})
