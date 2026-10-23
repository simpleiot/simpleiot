import FileIcon from "@lucide/svelte/icons/file"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.ProvisioningFile,
	label: "Provisioning File",
	icon: FileIcon,
	parents: [],
	ownedBy: NodeType.Provisioning,
	editor: Editor,
})
