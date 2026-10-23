// The signed-in user, kept in localStorage under the key and in the shape
// the UI at "/" uses, so one sign-in serves both UIs.

export interface User {
	token: string
	email: string
}

const storageKey = "storage"

function load(): User | null {
	try {
		const raw = localStorage.getItem(storageKey)
		if (!raw) {
			return null
		}
		const parsed: unknown = JSON.parse(raw)
		if (typeof parsed !== "object" || parsed === null || !("user" in parsed)) {
			return null
		}
		const { user } = parsed
		if (
			typeof user === "object" &&
			user !== null &&
			"token" in user &&
			"email" in user &&
			typeof user.token === "string" &&
			typeof user.email === "string" &&
			user.token !== ""
		) {
			return { token: user.token, email: user.email }
		}
	} catch {
		// unreadable storage is no user
	}
	return null
}

function save(user: User | null) {
	try {
		localStorage.setItem(storageKey, JSON.stringify({ user }))
	} catch {
		// storage may be unavailable; the session still works until reload
	}
}

class Auth {
	user = $state<User | null>(load())

	constructor() {
		// another tab, or the UI at "/", signing in or out
		window.addEventListener("storage", (e) => {
			if (e.key === storageKey || e.key === null) {
				this.user = load()
			}
		})
	}

	signIn(user: User) {
		save(user)
		this.user = user
	}

	signOut() {
		save(null)
		this.user = null
	}
}

export const auth = new Auth()
