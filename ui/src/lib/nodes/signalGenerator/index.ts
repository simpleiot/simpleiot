import ActivityIcon from "@lucide/svelte/icons/activity"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.SignalGenerator,
	label: "Signal Generator",
	icon: ActivityIcon,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "F",
	editor: Editor,
})
