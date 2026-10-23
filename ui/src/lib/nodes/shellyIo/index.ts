import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.ShellyIO,
	label: "Shelly IO",
	icon: Io,
	parents: [],
	ownedBy: NodeType.Shelly,
	sortKey: "Q",
})
