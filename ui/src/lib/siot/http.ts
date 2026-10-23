// The operations that go over HTTP. Each throws an HttpError with a
// readable message when the request fails.

import type { User } from "#lib/auth.svelte.ts"

export class HttpError extends Error {
	constructor(
		message: string,
		readonly status: number
	) {
		super(message)
	}
}

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
