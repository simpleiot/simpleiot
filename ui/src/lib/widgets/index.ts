// The registry of every widget in this directory. Adding a widget is adding
// a directory with an index.ts; nothing else names it.

import { createWidgetRegistry, type WidgetDescriptor } from "./registry.ts"

const modules = import.meta.glob<{ default: WidgetDescriptor }>("./*/index.ts", {
	eager: true,
})

export const widgets = createWidgetRegistry(Object.values(modules).map((m) => m.default))
