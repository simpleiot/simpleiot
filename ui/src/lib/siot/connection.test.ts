import { describe, expect, test, vi } from "vitest"
import type { Me, Message, SiotConnection, SiotSubscription } from "simpleiot-js"
import { Batcher, Connection, mergeNewest, type ConnectionEvents, type Env } from "./connection.ts"
import { DataType, type Point } from "./point.ts"

const pt = (type: string, value: number, ms: number): Point => ({
	type,
	key: "0",
	time: new Date(ms).toISOString(),
	dataType: DataType.Float,
	value,
	text: "",
	tombstone: 0,
})

describe("mergeNewest", () => {
	test("keeps the newest point per type and key", () => {
		const into = [pt("a", 1, 1000)]
		mergeNewest(into, [pt("a", 2, 2000), pt("a", 0, 500), pt("b", 3, 0)])
		expect(into.map((p) => [p.type, p.value])).toEqual([
			["a", 2],
			["b", 3],
		])
	})
})

describe("Batcher", () => {
	test("hands on one batch per frame, merged per node and edge", () => {
		const frames: (() => void)[] = []
		const onPoints = vi.fn()
		const onEdges = vi.fn()
		const b = new Batcher((f) => frames.push(f), onPoints, onEdges)
		b.queue({ nodeId: "n1", points: [pt("value", 1, 1000)] })
		b.queue({ nodeId: "n1", points: [pt("value", 2, 2000)] })
		b.queue({ nodeId: "n2", points: [pt("value", 3, 1000)] })
		b.queue({ nodeId: "n1", parentId: "p", points: [pt("tombstone", 1, 1000)] })
		expect(frames).toHaveLength(1)
		expect(onPoints).not.toHaveBeenCalled()

		frames[0]?.()
		expect(onPoints).toHaveBeenCalledWith([
			{ nodeId: "n1", points: [pt("value", 2, 2000)] },
			{ nodeId: "n2", points: [pt("value", 3, 1000)] },
		])
		expect(onEdges).toHaveBeenCalledWith([
			{ nodeId: "n1", parentId: "p", points: [pt("tombstone", 1, 1000)] },
		])

		b.queue({ nodeId: "n1", points: [pt("value", 4, 3000)] })
		expect(frames).toHaveLength(2)
	})
})

// fakeConn is a connection whose subscriptions yield what a test pushes.
function fakeConn() {
	const subs = new Map<string, { push: (m: Message) => void; sub: SiotSubscription }>()
	let close: (err?: Error) => void = () => undefined
	const closed = new Promise<Error | undefined>((resolve) => {
		close = (err) => {
			resolve(err)
		}
	})
	const me: Me = { userId: "u1", anchors: ["G"], nodes: [] }
	const conn = {
		me: () => Promise.resolve(me),
		closed: () => closed,
		close: () => {
			close()
			return Promise.resolve()
		},
		status: () => ({
			[Symbol.asyncIterator]: () => ({
				next: () => new Promise<IteratorResult<never>>(() => undefined),
			}),
		}),
		subscribe(subject: string) {
			const queue: Message[] = []
			let wake: (() => void) | null = null
			let done = false
			const sub = {
				unsubscribe: () => {
					done = true
					subs.delete(subject)
					wake?.()
				},
				getSubject: () => subject,
				async *[Symbol.asyncIterator]() {
					while (!done) {
						const m = queue.shift()
						if (m) {
							yield m
						} else {
							await new Promise<void>((r) => (wake = r))
						}
					}
				},
			} as unknown as SiotSubscription
			subs.set(subject, {
				push: (m) => {
					queue.push(m)
					wake?.()
				},
				sub,
			})
			return sub
		},
	} as unknown as SiotConnection
	return { conn, subs, close: (err?: Error) => close(err) }
}

function env(conn: SiotConnection, overrides: Partial<Env> = {}) {
	let visibility: () => void = () => undefined
	let hidden = false
	const frames: (() => void)[] = []
	const e: Env = {
		connect: () => Promise.resolve(conn),
		isAuthError: (err) => err instanceof Error && err.message === "auth",
		hidden: () => hidden,
		onVisibilityChange: (f) => {
			visibility = f
			return () => undefined
		},
		frame: (f) => frames.push(f),
		setTimeout: () => 1,
		clearTimeout: () => undefined,
		warn: () => undefined,
		...overrides,
	}
	return {
		env: e,
		frames,
		setHidden(h: boolean) {
			hidden = h
			visibility()
		},
	}
}

function events(): ConnectionEvents & { calls: string[] } {
	const calls: string[] = []
	return {
		calls,
		connected: (userId, anchors) => calls.push(`connected ${userId} ${anchors.join(",")}`),
		disconnected: (reason) => calls.push(`disconnected ${reason}`),
		authFailed: () => calls.push("authFailed"),
		points: (items) => calls.push(`points ${items.map((i) => i.nodeId).join(",")}`),
		edgePoints: (items) => calls.push(`edgePoints ${items.map((i) => i.nodeId).join(",")}`),
	}
}

const tick = () => new Promise((r) => setTimeout(r, 0))

describe("Connection", () => {
	test("announces, keeps subscriptions in step, and batches points", async () => {
		const f = fakeConn()
		const e = env(f.conn)
		const ev = events()
		const c = new Connection(ev, e.env)
		await c.connect("token")
		expect(ev.calls).toEqual(["connected u1 G"])

		c.watch(["up.G.a.*.*", "up.G.b.*.*"])
		expect([...f.subs.keys()]).toEqual(["up.G.a.*.*", "up.G.b.*.*"])
		c.watch(["up.G.b.*.*"])
		expect([...f.subs.keys()]).toEqual(["up.G.b.*.*"])

		f.subs
			.get("up.G.b.*.*")
			?.push({ subject: "", nodeId: "b", type: "value", key: "0", points: [pt("value", 1, 1)] })
		await tick()
		e.frames.shift()?.()
		expect(ev.calls).toContain("points b")
	})

	test("drops subscriptions while hidden and announces again when shown", async () => {
		const f = fakeConn()
		const e = env(f.conn)
		const ev = events()
		const c = new Connection(ev, e.env)
		await c.connect("token")
		c.watch(["up.G.a.*.*"])
		e.setHidden(true)
		expect(f.subs.size).toBe(0)
		c.watch(["up.G.a.*.*"])
		expect(f.subs.size).toBe(0)
		e.setHidden(false)
		await tick()
		expect(ev.calls.filter((s) => s.startsWith("connected"))).toHaveLength(2)
		expect(f.subs.size).toBe(1)
	})

	test("reports an auth failure, and retries other failures", async () => {
		const ev = events()
		const authFail = env({} as SiotConnection, { connect: () => Promise.reject(new Error("auth")) })
		await new Connection(ev, authFail.env).connect("t")
		expect(ev.calls).toEqual(["authFailed"])

		const retries: number[] = []
		const ev2 = events()
		const down = env({} as SiotConnection, {
			connect: () => Promise.reject(new Error("refused")),
			setTimeout: (_f, ms) => retries.push(ms),
		})
		await new Connection(ev2, down.env).connect("t")
		expect(ev2.calls).toEqual(["disconnected refused"])
		expect(retries).toEqual([5000])
	})
})
