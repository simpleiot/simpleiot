import { expect, test, vi } from "vitest"
import type { SiotConnection } from "simpleiot-js"
import type { Env } from "./connection.ts"
import type { Node } from "./node.ts"
import { getText, newText } from "./point.ts"
import { TreeStore } from "./tree.svelte.ts"

const group = (id: string, parent: string): Node => ({
	id,
	type: "group",
	parent,
	points: [newText("description", "0", id)],
	edgePoints: [],
})

// serve answers node requests from a list the way the server does: the
// nodes asked for, and depth more levels of their descendants.
function serve(nodes: Node[], parent: string, id: string, depth: number): Node[] {
	const below = (level: Node[], left: number): Node[] =>
		left < 0 || level.length === 0
			? []
			: [
					...level,
					...below(
						nodes.filter((n) => level.some((l) => l.id === n.parent)),
						left - 1
					),
				]
	return below(
		nodes.filter((n) => (parent === "all" ? n.id === id : n.parent === parent)),
		depth
	)
}

function env(nodes: Node[]): Env {
	const conn = {
		me: () => Promise.resolve({ userId: "u1", anchors: ["G"], nodes: [] }),
		closed: () => new Promise(() => undefined),
		close: () => Promise.resolve(),
		status: () => ({
			[Symbol.asyncIterator]: () => ({
				next: () => new Promise(() => undefined),
			}),
		}),
		subscribe: () => ({
			unsubscribe: () => undefined,
			[Symbol.asyncIterator]: () => ({
				next: () => new Promise(() => undefined),
			}),
		}),
		getNodes: (_anchor: string, parent: string, id: string, opts: { depth: number }) =>
			Promise.resolve(serve(nodes, parent, id, opts.depth)),
	} as unknown as SiotConnection
	return {
		connect: () => Promise.resolve(conn),
		isAuthError: () => false,
		hidden: () => false,
		onVisibilityChange: () => () => undefined,
		frame: (f) => {
			f()
		},
		setTimeout: () => 1,
		clearTimeout: () => undefined,
		warn: () => undefined,
	}
}

test("expanding shows changes made below it while it was collapsed", async () => {
	const nodes = [
		group("G", "root"),
		group("a", "G"),
		group("b", "a"),
		group("c", "b"),
		group("d", "c"),
	]
	const store = new TreeStore(env(nodes))
	store.start("token", () => undefined)
	await vi.waitFor(() => {
		expect(store.ready).toBe(true)
	})
	for (const key of ["G/G", "G/G/a", "G/G/a/b", "G/G/a/b/c"]) {
		await store.expand(key)
	}
	await store.toggleChildren("G/G/a")

	// nothing below a is watched now, so this change arrives only by a fetch
	const d = nodes[4] as Node
	nodes[4] = { ...d, points: [newText("description", "0", "renamed")] }
	await store.toggleChildren("G/G/a")

	const shown = store.find("G/G/a/b/c/d")
	expect(getText(shown?.node.points ?? [], "description")).toBe("renamed")
	await store.stop()
})
