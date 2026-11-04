import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

export default defineWidget({
	name: "value",
	label: "Value",
	host: "both",
	bindings: ["value"],
	options: { decimals: "1", unit: "", multiplier: "1", confirm: "" },
	component: Widget,
})
