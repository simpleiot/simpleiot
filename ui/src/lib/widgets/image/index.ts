import { PointType } from "#lib/siot/schema.ts"
import { defineWidget } from "../registry.ts"
import Widget from "./Widget.svelte"

// A picture uploaded as a file: the image binding names the file node,
// which holds a PNG, JPEG, GIF, or WebP uploaded as binary, or an SVG.
export default defineWidget({
	name: "image",
	label: "Image",
	host: "both",
	bindings: ["image"],
	options: { fit: "contain", opacity: "1" },
	width: 300,
	fit: "width",
	bindingPoints: { image: PointType.Data },
	component: Widget,
})
