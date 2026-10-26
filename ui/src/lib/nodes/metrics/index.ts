import ChartNoAxesColumnIcon from "@lucide/svelte/icons/chart-no-axes-column"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

export default defineNodeType({
	type: NodeType.Metrics,
	label: "Metrics",
	icon: ChartNoAxesColumnIcon,
	parents: [NodeType.Device],
	sortKey: "N",
	editor: Editor,
})
