import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Update,
	label: "Update",
	icon: RefreshCwIcon,
	parents: [NodeType.Device],
	sortKey: "T",
	editor: Editor,
})
