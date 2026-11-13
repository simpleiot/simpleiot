import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

// A slider over a range, for a setting such as a fan speed or a setpoint.
export default defineWidget({
	name: "slider",
	label: "Slider",
	host: "both",
	bindings: ["value"],
	options: {
		min: "0",
		max: "100",
		step: "1",
		unit: "",
		decimals: "0",
		multiplier: "1",
		confirm: "",
	},
	component: Widget,
})
