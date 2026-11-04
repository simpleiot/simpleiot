import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

// A nested dashboard: the binding names a dashboard node, and the frame
// draws that dashboard's widgets in place of this component.
export default defineWidget({
	name: "dashboard",
	label: "Nested dashboard",
	host: "both",
	bindings: ["dashboard"],
	options: {},
	width: 400,
	component: Widget,
})
