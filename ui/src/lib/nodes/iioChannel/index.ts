import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.IioChannel,
	label: "IIO Channel",
	icon: Io,
	parents: [NodeType.Iio],
	ownedBy: NodeType.Iio,
})
