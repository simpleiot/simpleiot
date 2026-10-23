import ChartNoAxesColumnIcon from "@lucide/svelte/icons/chart-no-axes-column"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"

export default defineNodeType({
	type: NodeType.Metrics,
	label: "Metrics",
	icon: ChartNoAxesColumnIcon,
	parents: [NodeType.Device],
	sortKey: "N",
})
