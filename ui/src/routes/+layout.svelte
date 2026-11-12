<script lang="ts">
	import "../app.css"
	import "#lib/vendor/nautilus/tokens.css"
	import { goto } from "$app/navigation"
	import { resolve } from "$app/paths"
	import { page } from "$app/state"
	import { auth } from "#lib/auth.svelte.ts"
	import SecurityNotice from "#lib/components/SecurityNotice.svelte"
	import { store } from "#lib/siot/store.ts"
	import { applyTheme } from "#lib/theme.svelte.ts"
	import type { Snippet } from "svelte"

	let { children }: { children: Snippet } = $props()

	applyTheme()

	const onSignIn = $derived(page.route.id === "/sign-in")

	// every page but sign-in needs a user, and a signed-in user has no use
	// for the sign-in page. The page asked for before sign-in is carried
	// in ?next, so a link to a dashboard or a widget still lands there.
	$effect(() => {
		if (!auth.user) {
			void store.stop()
		}
		if (!auth.user && !onSignIn) {
			const here = page.url.pathname + page.url.search
			const next = here === "/" ? "" : `?next=${encodeURIComponent(here)}`
			// the path is resolved; only the query is appended
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			void goto(`${resolve("/sign-in")}${next}`, { replaceState: true })
		} else if (auth.user && onSignIn) {
			// next is a path this app served, so it is resolved already
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			void goto(nextPath(page.url.searchParams.get("next")) ?? resolve("/"), {
				replaceState: true,
			})
		}
	})

	// nextPath accepts a path on this site only: one that starts with a
	// single slash
	function nextPath(next: string | null): string | null {
		return next !== null && next.startsWith("/") && !next.startsWith("//") ? next : null
	}
</script>

{#if onSignIn ? !auth.user : auth.user}
	{#if auth.user}
		<SecurityNotice user={auth.user} />
	{/if}
	{@render children()}
{/if}
