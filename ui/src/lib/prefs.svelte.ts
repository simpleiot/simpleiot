// What the UI remembers for itself, under its own localStorage key: the
// theme, and the expanded nodes for each user.

export type Theme = "light" | "dark"

export interface Expanded {
	anchor: string
	id: string
	// detail is set when the node's detail is open; otherwise its children
	// are expanded
	detail?: boolean
}

interface Stored {
	theme?: Theme
	expanded?: Record<string, Expanded[]>
}

const storageKey = "siot.ui"

function load(): Stored {
	try {
		const raw = localStorage.getItem(storageKey)
		const parsed: unknown = raw ? JSON.parse(raw) : {}
		if (typeof parsed === "object" && parsed !== null) {
			return parsed
		}
	} catch {
		// unreadable storage starts fresh
	}
	return {}
}

function save(s: Stored) {
	try {
		localStorage.setItem(storageKey, JSON.stringify(s))
	} catch {
		// storage may be unavailable; preferences then last for the session
	}
}

class Prefs {
	#stored = load()
	theme = $state<Theme | undefined>(this.#stored.theme)

	setTheme(theme: Theme | undefined) {
		this.theme = theme
		this.#stored.theme = theme
		save(this.#stored)
	}

	expanded(userId: string): Expanded[] {
		const list = this.#stored.expanded?.[userId]
		return Array.isArray(list) ? list : []
	}

	setExpanded(userId: string, list: Expanded[]) {
		this.#stored.expanded = { ...this.#stored.expanded, [userId]: list }
		save(this.#stored)
	}
}

export const prefs = new Prefs()
