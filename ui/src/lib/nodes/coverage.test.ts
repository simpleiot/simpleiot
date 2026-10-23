import { readdirSync, readFileSync, statSync } from "node:fs"
import { join, relative } from "node:path"
import { describe, expect, test } from "vitest"
import { NodeType } from "#lib/siot/schema.ts"
import { registry } from "./index.ts"

const src = join(import.meta.dirname, "../..")

function files(dir: string): string[] {
	return readdirSync(dir).flatMap((name) => {
		const p = join(dir, name)
		return statSync(p).isDirectory() ? files(p) : [p]
	})
}

describe("node types", () => {
	test("every node type has a descriptor with an editor", () => {
		const missing = Object.values(NodeType).filter((t) => !registry.editorFor(t))
		expect(missing).toEqual([])
		expect(registry.all.map((d) => d.type).sort()).toEqual(Object.values(NodeType).sort())
	})

	// The tree page and store ask the registry; only the node type
	// directories know types, so adding one is adding a directory. A type
	// name that steers behavior shows up as a comparison, a case label, or
	// a NodeType constant.
	test("no file outside src/lib/nodes names a node type", () => {
		const types = new Set<string>(Object.values(NodeType))
		const nodesDir = join(src, "lib/nodes")
		const offenders: string[] = []
		for (const f of files(src)) {
			if (
				f.startsWith(nodesDir) ||
				f.endsWith("schema.ts") ||
				f.endsWith(".test.ts") ||
				!/\.(ts|svelte)$/.test(f)
			) {
				continue
			}
			const text = readFileSync(f, "utf8")
			for (const m of text.matchAll(/(?:[!=]==?\s*|case\s+)(["'`])(\w+)\1/g)) {
				const literal = m[2] ?? ""
				if (types.has(literal)) {
					offenders.push(`${relative(src, f)}: ${literal}`)
				}
			}
			if (/NodeType\./.test(text)) {
				offenders.push(`${relative(src, f)}: NodeType`)
			}
		}
		expect(offenders).toEqual([])
	})
})
