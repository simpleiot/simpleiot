// The registry of every node type in this directory. Adding a type is adding
// a directory with an index.ts; nothing else names it.

import { createRegistry, type NodeTypeDescriptor } from "./registry.ts"

const modules = import.meta.glob<{ default: NodeTypeDescriptor }>("./*/index.ts", {
	eager: true,
})

export const registry = createRegistry(Object.values(modules).map((m) => m.default))
