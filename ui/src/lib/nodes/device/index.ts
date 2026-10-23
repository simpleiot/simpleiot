import HardDriveIcon from "@lucide/svelte/icons/hard-drive"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Device,
	label: "Device",
	icon: HardDriveIcon,
	parents: [],
	hasChildren: true,
	sortKey: "A",
})
