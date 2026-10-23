import Io from "#lib/icons/Io.svelte"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.MqttDevice,
	label: "MQTT Device",
	icon: Io,
	parents: [],
	sortKey: "W1",
	editor: Editor,
})
