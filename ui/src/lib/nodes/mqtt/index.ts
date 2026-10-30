import RssIcon from "@lucide/svelte/icons/rss"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Mqtt,
	label: "MQTT",
	icon: RssIcon,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "W",
	editor: Editor,
})
