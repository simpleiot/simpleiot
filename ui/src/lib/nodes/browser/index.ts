import GlobeIcon from "@lucide/svelte/icons/globe"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Browser,
	label: "Browser",
	icon: GlobeIcon,
	parents: [NodeType.Device],
	sortKey: "U",
})
