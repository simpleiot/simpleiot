import KeyIcon from "@lucide/svelte/icons/key"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.DeviceCred,
	label: "Device credential",
	icon: KeyIcon,
	parents: [NodeType.Device],
})
