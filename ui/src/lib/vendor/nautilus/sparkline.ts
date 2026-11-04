// Sparkline geometry — pure, dependency-free, and tested (tests/sparkline.test.ts).
//
// Extracted out of Sparkline.svelte so the one property that actually matters —
// this never emits a NaN into a path `d` attribute, no matter what `values`
// looks like — is something a spec can pin down directly, rather than
// something you have to trust from reading a $derived chain.

/** The vertical domain a series is plotted against. */
export interface SparklineDomain {
	lo: number
	hi: number
}

/**
 * `yMin`/`yMax` win when given (a fixed scale that does not jump as new
 * samples arrive); otherwise the domain is the series' own min/max. An
 * empty series (or one where `yMin`/`yMax` themselves are non-finite) falls
 * back to `[0, 1]` rather than propagating `Infinity`/`NaN` downstream. A
 * flat series (`hi - lo` below the epsilon) is widened by ±0.5 so the line
 * doesn't divide by zero.
 */
export function sparklineDomain(values: number[], yMin?: number, yMax?: number): SparklineDomain {
	let lo = yMin ?? Math.min(...values)
	let hi = yMax ?? Math.max(...values)
	if (!isFinite(lo) || !isFinite(hi)) return { lo: 0, hi: 1 }
	if (hi - lo < 1e-9) {
		lo -= 0.5
		hi += 0.5
	}
	return { lo, hi }
}

/** Inset from the box edge, and stroke width — denser in `compact` mode. */
export interface SparklineMetrics {
	xInset: number
	yInset: number
	strokeWidth: number
}

export function sparklineMetrics(compact = false): SparklineMetrics {
	return compact
		? { xInset: 1, yInset: 1, strokeWidth: 1 }
		: { xInset: 2, yInset: 3, strokeWidth: 1.5 }
}

export interface SparklinePoint {
	x: number
	y: number
}

export interface SparklineGeometry {
	/** SVG path `d`, or `''` when there is nothing to draw two points between. */
	path: string
	/** A single sample's position, when `values` has exactly one — drawn as a
	 *  dot rather than a path, since a line needs two points. `null` otherwise. */
	singlePoint: SparklinePoint | null
	/** Group transform-x, for scrolling mode; 0 in static mode. */
	tx: number
}

export interface SparklineGeometryOptions {
	width: number
	height: number
	yMin?: number
	yMax?: number
	compact?: boolean
	/** Absolute index of `values[values.length - 1]`; enables scrolling mode. */
	endIndex?: number
	/** Samples spanning the full width in scrolling mode (default: `values.length`). */
	windowSize?: number
}

/**
 * All the geometry `Sparkline.svelte` draws, computed once so it can be
 * unit-tested without a DOM. Never NaN: an empty series produces an empty
 * path and a null `singlePoint`; a one-sample series produces a dot instead
 * of a degenerate path; every division guards its denominator.
 */
export function sparklineGeometry(
	values: number[],
	opts: SparklineGeometryOptions
): SparklineGeometry {
	const { width, height, yMin, yMax, endIndex, windowSize, compact = false } = opts
	const { xInset, yInset } = sparklineMetrics(compact)
	const dom = sparklineDomain(values, yMin, yMax)
	const scrolling = endIndex !== undefined
	const end = endIndex ?? 0
	const span = Math.max((windowSize ?? values.length) - 1, 1)
	const dx = (width - 2 * xInset) / span
	const range = dom.hi - dom.lo || 1

	const yOf = (v: number) => yInset + (1 - (v - dom.lo) / range) * (height - 2 * yInset)

	let path = ""
	if (values.length > 1) {
		path = values
			.map((v, i) => {
				const x = scrolling
					? (end - values.length + 1 + i) * dx
					: (i / Math.max(values.length - 1, 1)) * (width - 2 * xInset) + xInset
				return `${i ? "L" : "M"} ${x.toFixed(2)} ${yOf(v).toFixed(2)}`
			})
			.join(" ")
	}

	// A single sample has no "position in time" to speak of in static mode —
	// center it rather than gluing it to the left edge. In scrolling mode it
	// keeps the same absolute placement a multi-point series would give it
	// (the newest sample sits at the leading edge as the window scrolls).
	const singlePoint: SparklinePoint | null =
		values.length === 1
			? {
					x: scrolling ? end * dx : width / 2,
					y: yOf(values[0] ?? 0),
				}
			: null

	const tx = scrolling ? width - xInset - end * dx : 0

	return { path, singlePoint, tx }
}
