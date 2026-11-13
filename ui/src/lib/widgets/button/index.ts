import { defineWidget, navigateBinding } from "../registry.ts"
import Widget from "./Widget.svelte"

// A button for moving between the screens of an HMI: an overview with a
// button per area, and each area with a button back. It binds the navigate
// binding every widget may take, or the url option, and the frame opens it.
export default defineWidget({
	name: "button",
	label: "Button",
	host: "both",
	bindings: [navigateBinding],
	options: { text: "" },
	width: 160,
	component: Widget,
})
