import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Sync,
	label: "Sync",
	icon: RefreshCwIcon,
	parents: [NodeType.Device],
	editor: Editor,
})
