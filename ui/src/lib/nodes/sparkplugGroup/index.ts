import RssIcon from "@lucide/svelte/icons/rss"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.SparkplugGroup,
	label: "Sparkplug Group",
	icon: RssIcon,
	parents: [],
	ownedBy: NodeType.Mqtt,
	sortKey: "X",
})
