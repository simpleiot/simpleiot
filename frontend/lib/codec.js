// The binary encoding of points and node replies, mirroring data/point.go
// and data/node.go in the Go code. Integers are little endian, strings and
// byte fields carry a two-byte length, time is int64 nanoseconds since the
// epoch, and a node reply starts with a version byte and an error string.
//
// Points are returned in the shape the web UI reads:
//   { type, key, time, dataType, value, text, tombstone, origin }
// with time as an ISO 8601 string, value the numeric value (0 for text
// points), and text the string value ("" for numeric points).
//
// The Go tests in data/point_fixture_test.go write the fixtures in testdata/
// that test.js checks this file against, so the two encoders cannot drift
// without a test failing.

const TE = new TextEncoder()
const TD = new TextDecoder()

// data types, as in data/point.go
export const DataType = {
	Unknown: 0,
	Float: 1,
	Int: 2,
	String: 3,
	JSON: 4,
}

const nodeFrameVersion = 1
const maxPointsPerMessage = 10000
const maxNodesPerFrame = 10000

class Writer {
	constructor() {
		this.chunks = []
		this.length = 0
	}

	push(bytes) {
		this.chunks.push(bytes)
		this.length += bytes.length
	}

	u8(v) {
		this.push(Uint8Array.of(v & 0xff))
	}

	u16(v) {
		const b = new Uint8Array(2)
		new DataView(b.buffer).setUint16(0, v, true)
		this.push(b)
	}

	u32(v) {
		const b = new Uint8Array(4)
		new DataView(b.buffer).setUint32(0, v >>> 0, true)
		this.push(b)
	}

	i64(v) {
		const b = new Uint8Array(8)
		new DataView(b.buffer).setBigInt64(0, BigInt(v), true)
		this.push(b)
	}

	bytes(b) {
		if (b.length > 0xffff) {
			throw new Error(`field of ${b.length} bytes is too long to encode`)
		}
		this.u16(b.length)
		this.push(b)
	}

	string(s) {
		this.bytes(TE.encode(s || ""))
	}

	result() {
		const out = new Uint8Array(this.length)
		let off = 0
		for (const c of this.chunks) {
			out.set(c, off)
			off += c.length
		}
		return out
	}
}

class Reader {
	constructor(data) {
		this.data = data
		this.view = new DataView(data.buffer, data.byteOffset, data.byteLength)
		this.off = 0
	}

	need(n, what) {
		if (this.off + n > this.data.length) {
			throw new Error(
				`decode: not enough data for ${what} at offset ${this.off}`
			)
		}
	}

	u8() {
		this.need(1, "byte")
		return this.data[this.off++]
	}

	u16() {
		this.need(2, "length")
		const v = this.view.getUint16(this.off, true)
		this.off += 2
		return v
	}

	u32() {
		this.need(4, "count")
		const v = this.view.getUint32(this.off, true)
		this.off += 4
		return v
	}

	i32() {
		this.need(4, "int32")
		const v = this.view.getInt32(this.off, true)
		this.off += 4
		return v
	}

	i64() {
		this.need(8, "int64")
		const v = this.view.getBigInt64(this.off, true)
		this.off += 8
		return v
	}

	bytes() {
		const n = this.u16()
		this.need(n, "bytes")
		const b = this.data.subarray(this.off, this.off + n)
		this.off += n
		return b
	}

	string() {
		return TD.decode(this.bytes())
	}
}

// timeToNanos converts a Date, ISO string, or millisecond number to
// nanoseconds since the epoch. A missing time is now.
function timeToNanos(t) {
	let ms
	if (t == null || t === "") {
		ms = Date.now()
	} else if (t instanceof Date) {
		ms = t.valueOf()
	} else if (typeof t === "number") {
		ms = t
	} else {
		ms = new Date(t).valueOf()
	}
	if (Number.isNaN(ms)) {
		throw new Error(`invalid point time ${t}`)
	}
	return BigInt(Math.round(ms)) * 1000000n
}

function nanosToISO(ns) {
	const ms = Number(ns / 1000000n)
	return new Date(ms).toISOString()
}

// intBytes sizes an integer the way data.Point.PutInt does: 1, 2, 4, or 8
// bytes by magnitude.
function intBytes(v) {
	const abs = Math.abs(v)
	let size
	if (abs < 128) {
		size = 1
	} else if (abs < 32768) {
		size = 2
	} else if (abs < 2147483648) {
		size = 4
	} else {
		size = 8
	}
	const b = new Uint8Array(size)
	const dv = new DataView(b.buffer)
	switch (size) {
		case 1:
			b[0] = v & 0xff
			break
		case 2:
			dv.setInt16(0, v, true)
			break
		case 4:
			dv.setInt32(0, v, true)
			break
		default:
			dv.setBigInt64(0, BigInt(Math.trunc(v)), true)
	}
	return b
}

// pointData works out the data type and bytes for a point given in the UI
// shape. The rule is the one the HTTP API applied to JSON points: a text
// value makes a string point, otherwise a non-zero value makes a float
// point, otherwise the point carries no data. A dataType of JSON or Int is
// honored when the matching field is set, and raw data is passed through
// when the point carries it.
function pointData(p) {
	const { text = "", value = 0, dataType = DataType.Unknown, data } = p
	if (text !== "") {
		const dt = dataType === DataType.JSON ? DataType.JSON : DataType.String
		return [dt, TE.encode(text)]
	}
	if (value !== 0) {
		if (dataType === DataType.Int && Number.isInteger(value)) {
			return [DataType.Int, intBytes(value)]
		}
		const b = new Uint8Array(8)
		new DataView(b.buffer).setFloat64(0, value, true)
		return [DataType.Float, b]
	}
	if (data instanceof Uint8Array && data.length > 0) {
		return [dataType, data]
	}
	return [DataType.Unknown, new Uint8Array(0)]
}

function encodePoint(w, p) {
	const [dataType, data] = pointData(p)
	w.string(p.type)
	w.string(p.key)
	w.i64(timeToNanos(p.time))
	w.u8(dataType)
	w.bytes(data)
	w.u32(p.tombstone || 0)
	w.string(p.origin)
}

// decodeValue reads the numeric value the way data.Point.Val does.
function decodeValue(dataType, data) {
	const dv = new DataView(data.buffer, data.byteOffset, data.byteLength)
	if (dataType === DataType.Float) {
		if (data.length === 4) {
			return dv.getFloat32(0, true)
		}
		if (data.length === 8) {
			return dv.getFloat64(0, true)
		}
		return 0
	}
	if (dataType === DataType.Int) {
		switch (data.length) {
			case 1:
				return data[0]
			case 2:
				return dv.getUint16(0, true)
			case 4:
				return dv.getUint32(0, true)
			case 8:
				return Number(dv.getBigInt64(0, true))
			default:
				return 0
		}
	}
	return 0
}

function decodePoint(r) {
	const type = r.string()
	const key = r.string()
	const time = nanosToISO(r.i64())
	const dataType = r.u8()
	const data = r.bytes()
	const tombstone = r.i32()
	const origin = r.string()
	const text =
		dataType === DataType.String || dataType === DataType.JSON
			? TD.decode(data)
			: ""
	return {
		type,
		key,
		time,
		dataType,
		value: decodeValue(dataType, data),
		text,
		tombstone,
		origin,
	}
}

function encodePointsTo(w, points) {
	w.u32(points.length)
	for (const p of points) {
		encodePoint(w, p)
	}
}

function decodePointsFrom(r) {
	const count = r.u32()
	if (count > maxPointsPerMessage) {
		throw new Error(`decode: point count ${count} exceeds maximum`)
	}
	const points = []
	for (let i = 0; i < count; i++) {
		points.push(decodePoint(r))
	}
	return points
}

// encodePoints returns the wire encoding of an array of points.
export function encodePoints(points) {
	const w = new Writer()
	encodePointsTo(w, points)
	return w.result()
}

// decodePoints returns the points in a wire payload.
export function decodePoints(data) {
	return decodePointsFrom(new Reader(data))
}

// decodeNodes returns the nodes in a node reply, each as
// { id, type, parent, points, edgePoints }. An error the sender put in the
// frame is thrown. An empty payload is no nodes.
export function decodeNodes(data) {
	if (!data || data.length === 0) {
		return []
	}
	const r = new Reader(data)
	const version = r.u8()
	if (version !== nodeFrameVersion) {
		throw new Error(`decode: unsupported node frame version ${version}`)
	}
	const err = r.string()
	if (err !== "") {
		throw new Error(err)
	}
	const count = r.u32()
	if (count > maxNodesPerFrame) {
		throw new Error(`decode: node count ${count} exceeds maximum`)
	}
	const nodes = []
	for (let i = 0; i < count; i++) {
		const id = r.string()
		const type = r.string()
		const parent = r.string()
		const points = decodePointsFrom(r)
		const edgePoints = decodePointsFrom(r)
		nodes.push({ id, type, parent, points, edgePoints })
	}
	return nodes
}
