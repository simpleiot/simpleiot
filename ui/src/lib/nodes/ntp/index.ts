import ClockIcon from "@lucide/svelte/icons/clock"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.NTP,
	label: "NTP",
	icon: ClockIcon,
	parents: [NodeType.Device],
	sortKey: "S",
})
