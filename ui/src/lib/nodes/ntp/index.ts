import ClockIcon from "@lucide/svelte/icons/clock"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.NTP,
	label: "NTP",
	icon: ClockIcon,
	parents: [NodeType.Device],
	sortKey: "S",
	editor: Editor,
})
