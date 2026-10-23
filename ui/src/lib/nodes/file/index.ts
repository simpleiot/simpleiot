import FileIcon from "@lucide/svelte/icons/file"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.File,
	label: "File",
	icon: FileIcon,
	parents: [
		NodeType.Device,
		NodeType.Group,
		NodeType.CanBus,
		NodeType.SerialDev,
		NodeType.Provisioning,
	],
	sortKey: "K",
})
