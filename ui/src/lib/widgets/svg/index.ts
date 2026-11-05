import { PointType } from "#lib/siot/schema.ts"
import { defineWidget } from "../registry.ts"
import { parseDrawing } from "./drawing.ts"
import Widget from "./Widget.svelte"

// A drawing uploaded as a file: the svg binding names the file node, and
// the drawing's own data-siot-* attributes declare the bindings and ports
// the rest of the widget takes.
export default defineWidget({
	name: "svg",
	label: "Drawing (SVG)",
	host: "both",
	bindings: ["svg"],
	options: { confirm: "" },
	width: 200,
	fit: "width",
	bindingPoints: { svg: PointType.Data },
	source: "svg",
	bindingsIn: (text) => parseDrawing(text).bindings,
	portsIn: (text) => parseDrawing(text).ports,
	component: Widget,
})
