import KeyIcon from "@lucide/svelte/icons/key"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.DeviceCred,
	label: "Device credential",
	icon: KeyIcon,
	parents: [NodeType.Device],
	editor: Editor,
})
