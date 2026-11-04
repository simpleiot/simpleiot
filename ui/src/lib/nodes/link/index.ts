import SplineIcon from "@lucide/svelte/icons/spline"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Link,
	label: "Link",
	icon: SplineIcon,
	parents: [NodeType.Dashboard],
	ownedBy: NodeType.Dashboard,
	sortKey: "B",
	editor: Editor,
})
