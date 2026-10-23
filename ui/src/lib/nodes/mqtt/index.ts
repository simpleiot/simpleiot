import RssIcon from "@lucide/svelte/icons/rss"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Mqtt,
	label: "MQTT",
	icon: RssIcon,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "W",
})
