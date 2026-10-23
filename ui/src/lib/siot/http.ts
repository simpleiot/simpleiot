// The node operations that go over HTTP, with the request bodies the API
// takes (see api/nodes.go). Each throws an Error with a readable message
// when the request fails.

import type { User } from "#lib/auth.svelte.ts"
import type { Node } from "./node.ts"
import type { Point } from "./point.ts"

export class HttpError extends Error {
	constructor(
		message: string,
		readonly status: number
	) {
		super(message)
	}
}

interface Response {
	success: boolean
	error?: string
	id?: string
}

async function request<T>(method: string, url: string, token: string, body?: unknown): Promise<T> {
	let res: globalThis.Response
	try {
		res = await fetch(url, {
			method,
			headers: {
				Authorization: `Bearer ${token}`,
				...(body === undefined ? {} : { "Content-Type": "application/json" }),
			},
			body: body === undefined ? undefined : JSON.stringify(body),
		})
	} catch {
		throw new HttpError("Network error", 0)
	}
	if (!res.ok) {
		throw new HttpError(`Bad status: ${res.status}`, res.status)
	}
	const text = await res.text()
	try {
		return JSON.parse(text) as T
	} catch {
		throw new HttpError(`Bad body: ${text}`, res.status)
	}
}

async function op(method: string, url: string, token: string, body?: unknown): Promise<Response> {
	const r = await request<Response>(method, url, token, body)
	if (r.error) {
		throw new HttpError(r.error, 200)
	}
	return r
}

const nodeUrl = (id: string, rest = "") => `/v1/nodes/${encodeURIComponent(id)}${rest}`

// signIn exchanges an email and password for a token. An invalid login is
// returned as null.
export async function signIn(email: string, password: string): Promise<User | null> {
	const form = new FormData()
	form.append("email", email)
	form.append("password", password)
	let res: globalThis.Response
	try {
		res = await fetch("/v1/auth", { method: "POST", body: form })
	} catch {
		throw new HttpError("Network error", 0)
	}
	if (res.status === 403) {
		return null
	}
	if (!res.ok) {
		throw new HttpError(`Bad status: ${res.status}`, res.status)
	}
	const user = (await res.json()) as Partial<User>
	if (typeof user.token !== "string" || user.token === "") {
		return null
	}
	return { token: user.token, email: typeof user.email === "string" ? user.email : email }
}

// insertNode adds a node under node.parent.
export function insertNode(token: string, node: Node): Promise<Response> {
	return op("POST", nodeUrl(node.id), token, {
		id: node.id,
		type: node.type,
		hash: 0,
		parent: node.parent,
		points: node.points.map(toJSON),
		edgePoints: node.edgePoints.map(toJSON),
	})
}

// deleteNode removes the edge from a node to its parent.
export function deleteNode(token: string, id: string, parent: string): Promise<Response> {
	return op("DELETE", nodeUrl(id), token, { parent })
}

export function moveNode(
	token: string,
	id: string,
	oldParent: string,
	newParent: string
): Promise<Response> {
	return op("POST", nodeUrl(id, "/parents"), token, { id, oldParent, newParent })
}

// copyNode mirrors a node under a new parent, or with duplicate set makes a
// new copy of it there.
export function copyNode(
	token: string,
	id: string,
	oldParent: string,
	newParent: string,
	duplicate: boolean
): Promise<Response> {
	return op("PUT", nodeUrl(id, "/parents"), token, {
		id,
		oldParent,
		newParent,
		duplicate,
	})
}

export function notify(token: string, sourceNode: string, message: string): Promise<Response> {
	return op("POST", nodeUrl(sourceNode, "/not"), token, {
		id: "",
		sourceNode,
		subject: "",
		message,
	})
}

// generateKey makes a token for an enrollToken node. The node stores its
// hash; the token itself is returned once.
export async function generateKey(token: string, id: string): Promise<string> {
	const r = await request<{ token?: string }>("POST", nodeUrl(id, "/key"), token)
	if (typeof r.token !== "string") {
		throw new HttpError("Bad body: no token", 200)
	}
	return r.token
}

function toJSON(p: Point) {
	return {
		type: p.type,
		key: p.key,
		time: p.time,
		dataType: p.dataType,
		value: p.value,
		text: p.text,
		tombstone: p.tombstone,
	}
}
