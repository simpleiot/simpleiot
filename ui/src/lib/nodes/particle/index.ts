import Particle from "#lib/icons/Particle.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Particle,
	label: "Particle",
	icon: Particle,
	parents: [NodeType.Device, NodeType.Group],
	sortKey: "O",
	editor: Editor,
})
