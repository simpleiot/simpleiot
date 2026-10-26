import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.OneWireIO,
	label: "1-Wire IO",
	icon: Io,
	parents: [],
	ownedBy: NodeType.OneWire,
	editor: Editor,
})
