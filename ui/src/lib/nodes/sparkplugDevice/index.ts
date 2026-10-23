import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.SparkplugDevice,
	label: "Sparkplug Device",
	icon: Io,
	parents: [],
	ownedBy: NodeType.SparkplugNode,
	sortKey: "Z",
})
