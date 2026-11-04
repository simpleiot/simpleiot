import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "trend",
	label: "Trend",
	host: "both",
	bindings: ["value"],
	options: { window: "120", min: "", max: "", unit: "", decimals: "1" },
	component: Widget,
})
