import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.ShellyIO,
	label: "Shelly IO",
	icon: Io,
	parents: [],
	ownedBy: NodeType.Shelly,
	sortKey: "Q",
	editor: Editor,
})
