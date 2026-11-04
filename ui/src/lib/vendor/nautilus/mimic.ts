// Ports, pipe routing, and anchoring, from nautilus's mimic.ts: the parts
// the canvas host draws links with. A port is a named point on an
// equipment's rendered box, as fractions of it; a link end anchored to a
// port is derived from the box each render, so moving the equipment moves
// the link. The editor-only helpers (attaching free ends, auto-connecting)
// are left out.

/** The direction a pipe LEAVES a port, in canvas space. */
export type PortDir = "left" | "right" | "up" | "down"

/** A named connection point, as a fraction of the rendered box. */
export interface MimicPort {
	name: string
	x: number
	y: number
	dir?: PortDir
}

/** A port's exit direction, inferred from its position on the box's edge:
 * exactly one axis must be pinned to an edge; a corner or interior port
 * has none. */
export function inferPortDir(port: Pick<MimicPort, "x" | "y">): PortDir | undefined {
	const xEdge = port.x === 0 || port.x === 1
	const yEdge = port.y === 0 || port.y === 1
	if (xEdge === yEdge) {
		return undefined
	}
	return xEdge ? (port.x === 0 ? "left" : "right") : port.y === 0 ? "up" : "down"
}

/** The explicit direction when set, else the inferred one. */
export function resolvedPortDir(port: Pick<MimicPort, "x" | "y" | "dir">): PortDir | undefined {
	return port.dir ?? inferPortDir(port)
}

export type Point = [number, number]

/** Polyline points -> SVG path ("M x y L x y ..."). */
export function pointsToPath(points: readonly Point[]): string {
	return points.map(([x, y], i) => `${i === 0 ? "M" : "L"} ${x} ${y}`).join(" ")
}

/** Expand a point list into an orthogonal (90 degree) route: each
 * consecutive pair gets one corner between them instead of a diagonal.
 * Each segment continues the previous segment's arrival axis first, then
 * turns; the first segment picks the axis with the larger delta. A pair
 * already axis-aligned passes through as one straight leg. */
export function orthogonalPoints(points: readonly Point[]): Point[] {
	const first = points[0]
	if (!first || points.length < 2) {
		return points.slice()
	}
	const out: Point[] = [first]
	let lastAxis: "h" | "v" | null = null
	for (let i = 1; i < points.length; i++) {
		const prev = points[i - 1]
		const cur = points[i]
		if (!prev || !cur) {
			continue
		}
		const [x0, y0] = prev
		const [x1, y1] = cur
		if (x0 === x1 || y0 === y1) {
			out.push([x1, y1])
			if (x0 !== x1 || y0 !== y1) {
				lastAxis = x0 === x1 ? "v" : "h"
			}
			continue
		}
		const firstAxis: "h" | "v" = lastAxis ?? (Math.abs(x1 - x0) >= Math.abs(y1 - y0) ? "h" : "v")
		out.push(firstAxis === "h" ? [x1, y0] : [x0, y1])
		out.push([x1, y1])
		lastAxis = firstAxis === "h" ? "v" : "h"
	}
	return out
}

/** An equipment's rendered box in canvas units: top-left plus measured
 * size. */
export interface EquipmentBox {
	x: number
	y: number
	w: number
	h: number
}

/** A port resolved to canvas units, with its exit direction. */
export interface ResolvedPort {
	x: number
	y: number
	dir?: PortDir
}

/** A port fraction in a box -> its canvas position and direction. */
export function portAbsolute(
	box: EquipmentBox,
	port: Pick<MimicPort, "x" | "y" | "dir">
): ResolvedPort {
	return { x: box.x + port.x * box.w, y: box.y + port.y * box.h, dir: resolvedPortDir(port) }
}

/** Looks up one equipment's named port in canvas units, or undefined when
 * the equipment or the port does not exist. */
export type GetPort = (equip: string, port: string) => ResolvedPort | undefined

export function makeGetPort(
	getBox: (equip: string) => EquipmentBox | undefined,
	getPorts: (equip: string) => readonly MimicPort[] | undefined
): GetPort {
	return (equip, port) => {
		const box = getBox(equip)
		if (!box) {
			return undefined
		}
		const p = (getPorts(equip) ?? []).find((q) => q.name === port)
		return p ? portAbsolute(box, p) : undefined
	}
}

export interface PipeAnchor {
	equip: string
	port: string
}

export interface PipeSpec {
	/** Interior vertices only when an end is anchored; the full list
	 * otherwise. */
	points: readonly Point[]
	routing?: "direct" | "orthogonal"
	from?: PipeAnchor
	to?: PipeAnchor
}

export interface PipeEndpoints {
	points: Point[]
	startDir?: PortDir
	endDir?: PortDir
	/** An anchor that did not resolve; the geometry still comes back
	 * complete, from the nearest vertex or the other end. */
	startFlagged: boolean
	endFlagged: boolean
}

/** Resolve a pipe's anchors and splice them onto the interior points. An
 * unresolved anchor falls back to the nearest interior vertex, or the
 * other end, or the origin, so a pipe never disappears. */
export function resolvePipeEndpoints(pipe: PipeSpec, getPort: GetPort): PipeEndpoints {
	const interior: Point[] = pipe.points.map((p) => [p[0], p[1]])
	let startFlagged = false
	let endFlagged = false
	let startDir: PortDir | undefined
	let endDir: PortDir | undefined
	let startPoint: Point | undefined
	let endPoint: Point | undefined

	if (pipe.from) {
		const r = getPort(pipe.from.equip, pipe.from.port)
		if (r) {
			startPoint = [r.x, r.y]
			startDir = r.dir
		} else {
			startFlagged = true
		}
	}
	if (pipe.to) {
		const r = getPort(pipe.to.equip, pipe.to.port)
		if (r) {
			endPoint = [r.x, r.y]
			endDir = r.dir
		} else {
			endFlagged = true
		}
	}

	const points: Point[] = []
	if (startPoint) {
		points.push(startPoint)
	}
	points.push(...interior)
	if (endPoint) {
		points.push(endPoint)
	}
	if (pipe.from && !startPoint) {
		points.unshift(points[0] ?? endPoint ?? [0, 0])
	}
	if (pipe.to && !endPoint) {
		points.push(points[points.length - 1] ?? startPoint ?? [0, 0])
	}
	return { points, startDir, endDir, startFlagged, endFlagged }
}

/** The exit stub a pipe draws leaving a directional port, in canvas units. */
export const PORT_STUB = 12

const DIR_VECTOR: Record<PortDir, Point> = {
	left: [-1, 0],
	right: [1, 0],
	up: [0, -1],
	down: [0, 1],
}

function withDirStub(points: Point[], dir: PortDir | undefined, atStart: boolean): Point[] {
	const anchor = atStart ? points[0] : points[points.length - 1]
	const next = atStart ? points[1] : points[points.length - 2]
	if (!dir || points.length < 2 || !anchor || !next) {
		return points
	}
	const [dx, dy] = DIR_VECTOR[dir]
	const stub: Point = [anchor[0] + dx * PORT_STUB, anchor[1] + dy * PORT_STUB]
	if (next[0] === stub[0] && next[1] === stub[1]) {
		return points
	}
	return atStart ? [anchor, stub, ...points.slice(1)] : [...points.slice(0, -1), stub, anchor]
}

/** The points to draw for a pipe: anchors resolved, exit stubs added at
 * anchored ends that have a direction, then routed orthogonally when the
 * pipe asks for it. */
export function routedPoints(pipe: PipeSpec, getPort?: GetPort): Point[] {
	let pts: Point[]
	let startDir: PortDir | undefined
	let endDir: PortDir | undefined
	if (pipe.from || pipe.to) {
		const r = resolvePipeEndpoints(pipe, getPort ?? (() => undefined))
		pts = r.points
		startDir = r.startDir
		endDir = r.endDir
	} else {
		pts = pipe.points.map((p) => [p[0], p[1]])
	}
	pts = withDirStub(pts, startDir, true)
	pts = withDirStub(pts, endDir, false)
	return pipe.routing === "orthogonal" ? orthogonalPoints(pts) : pts
}
