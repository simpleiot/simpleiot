import RadioReceiverIcon from "@lucide/svelte/icons/radio-receiver"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.SparkplugNode,
	label: "Sparkplug Node",
	icon: RadioReceiverIcon,
	parents: [],
	ownedBy: NodeType.SparkplugGroup,
	sortKey: "Y",
})
