// Applies the theme preference to the page. With no preference the page
// follows the system setting through prefers-color-scheme in app.css.

import { prefs, type Theme } from "./prefs.svelte.ts"

export function applyTheme() {
	$effect(() => {
		const root = document.documentElement
		if (prefs.theme) {
			root.dataset.theme = prefs.theme
		} else {
			delete root.dataset.theme
		}
	})
}

// effectiveTheme is the theme on screen, preference or system.
export function effectiveTheme(): Theme {
	if (prefs.theme) {
		return prefs.theme
	}
	return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
}

export function toggleTheme() {
	const next: Theme = effectiveTheme() === "dark" ? "light" : "dark"
	const system: Theme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
	// toggling back to what the system shows clears the override
	prefs.setTheme(next === system ? undefined : next)
}
