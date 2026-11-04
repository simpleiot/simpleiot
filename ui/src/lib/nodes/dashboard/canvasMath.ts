// Canvas geometry: the plane is drawn in canvas units, scaled to fit the
// frame, then zoomed and panned by the viewer. These helpers keep the
// mapping between the two in one place.

export interface View {
	// scale is canvas units to frame pixels, fit times zoom
	scale: number
	// panX and panY are the plane's offset in frame pixels
	panX: number
	panY: number
}

// fitScale draws the whole plane across the frame's width.
export function fitScale(frameWidth: number, canvasWidth: number): number {
	return frameWidth > 0 && canvasWidth > 0 ? frameWidth / canvasWidth : 1
}

// toCanvas maps a frame position to canvas units.
export function toCanvas(fx: number, fy: number, v: View): [number, number] {
	return [(fx - v.panX) / v.scale, (fy - v.panY) / v.scale]
}

// zoomAt changes the scale by a factor about a frame position, so the
// point under the pointer stays put. The scale is kept within bounds
// relative to the fit scale.
export function zoomAt(v: View, factor: number, fx: number, fy: number, fit: number): View {
	const scale = Math.min(fit * 8, Math.max(fit * 0.25, v.scale * factor))
	const k = scale / v.scale
	return { scale, panX: fx - (fx - v.panX) * k, panY: fy - (fy - v.panY) * k }
}

// canvasDelta turns a pointer movement in frame pixels into canvas units,
// rounded to whole units.
export function canvasDelta(dx: number, dy: number, v: View): [number, number] {
	return [Math.round(dx / v.scale), Math.round(dy / v.scale)]
}

// parsePair reads "x,y" into numbers, or undefined when it is not one.
export function parsePair(text: string): [number, number] | undefined {
	const parts = text.split(",").map((s) => Number(s.trim()))
	const [x, y] = parts
	return parts.length === 2 &&
		x !== undefined &&
		y !== undefined &&
		Number.isFinite(x) &&
		Number.isFinite(y)
		? [x, y]
		: undefined
}
