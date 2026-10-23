import MapPinIcon from "@lucide/svelte/icons/map-pin"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Gps,
	label: "GPS",
	icon: MapPinIcon,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "V",
})
