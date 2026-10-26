import MapPinIcon from "@lucide/svelte/icons/map-pin"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Gps,
	label: "GPS",
	icon: MapPinIcon,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "V",
	editor: Editor,
})
