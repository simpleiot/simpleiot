import KeyIcon from "@lucide/svelte/icons/key"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.EnrollToken,
	label: "Enrollment token",
	icon: KeyIcon,
	parents: [NodeType.Device, NodeType.Group],
})
