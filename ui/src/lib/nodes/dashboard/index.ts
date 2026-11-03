import LayoutDashboardIcon from "@lucide/svelte/icons/layout-dashboard"
import { NodeType } from "#lib/siot/schema.ts"
import { defineNodeType } from "../registry.ts"
import Editor from "./Editor.svelte"

// dashboardType is exported for the dashboard pages, which live outside
// this directory and so name no type themselves.
export const dashboardType = NodeType.Dashboard

export default defineNodeType({
	type: NodeType.Dashboard,
	label: "Dashboard",
	icon: LayoutDashboardIcon,
	parents: [NodeType.Device, NodeType.Group],
	hasChildren: true,
	sortKey: "D",
	editor: Editor,
})
