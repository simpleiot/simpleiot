import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.ModbusIO,
	label: "Modbus IO",
	icon: Io,
	parents: [NodeType.Modbus],
	ownedBy: NodeType.Modbus,
	editor: Editor,
})
