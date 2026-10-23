import OneWire from "#lib/icons/OneWire.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.OneWire,
	label: "1-Wire",
	icon: OneWire,
	parents: [],
	hasChildren: true,
	sortKey: "G",
})
