import HardDriveIcon from "@lucide/svelte/icons/hard-drive"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Device,
	label: "Device",
	icon: HardDriveIcon,
	parents: [],
	hasChildren: true,
	sortKey: "A",
	editor: Editor,
})
