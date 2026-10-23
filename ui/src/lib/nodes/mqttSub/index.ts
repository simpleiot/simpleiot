import HashIcon from "@lucide/svelte/icons/hash"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.MqttSub,
	label: "MQTT Subscription",
	icon: HashIcon,
	parents: [NodeType.Mqtt],
	ownedBy: NodeType.Mqtt,
	editor: Editor,
})
