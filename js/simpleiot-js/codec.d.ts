// Types for codec.js, the binary encoding of points and node replies.

export declare const DataType: {
	readonly Unknown: 0
	readonly Float: 1
	readonly Int: 2
	readonly String: 3
	readonly JSON: 4
}

// Point is a point as the codec reads it: time is an ISO 8601 string,
// value is 0 for text points, and text is "" for numeric ones.
export interface Point {
	type: string
	key: string
	time: string
	dataType: number
	value: number
	text: string
	tombstone: number
	origin?: string
}

// PointInput is a point to encode. A text value makes a string point,
// otherwise a non-zero value makes a number point; a missing time is now.
export interface PointInput {
	type: string
	key?: string
	time?: string | number | Date
	dataType?: number
	value?: number
	text?: string
	tombstone?: number
	origin?: string
	data?: Uint8Array
}

export interface Node {
	id: string
	type: string
	parent: string
	points: Point[]
	edgePoints: Point[]
}

export declare function encodePoints(points: readonly PointInput[]): Uint8Array
export declare function decodePoints(data: Uint8Array): Point[]
export declare function decodeNodes(data: Uint8Array): Node[]
