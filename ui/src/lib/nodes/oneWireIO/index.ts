import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.OneWireIO,
	label: "1-Wire IO",
	icon: Io,
	parents: [],
	ownedBy: NodeType.OneWire,
})
