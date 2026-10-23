<script lang="ts">
	import "../app.css"
	import { goto } from "$app/navigation"
	import { resolve } from "$app/paths"
	import { page } from "$app/state"
	import { auth } from "#lib/auth.svelte.ts"
	import { store } from "#lib/siot/store.ts"
	import { applyTheme } from "#lib/theme.svelte.ts"
	import type { Snippet } from "svelte"

	let { children }: { children: Snippet } = $props()

	applyTheme()

	const onSignIn = $derived(page.route.id === "/sign-in")

	// every page but sign-in needs a user, and a signed-in user has no use
	// for the sign-in page
	$effect(() => {
		if (!auth.user) {
			void store.stop()
		}
		if (!auth.user && !onSignIn) {
			void goto(resolve("/sign-in"), { replaceState: true })
		} else if (auth.user && onSignIn) {
			void goto(resolve("/"), { replaceState: true })
		}
	})
</script>

{#if onSignIn ? !auth.user : auth.user}
	{@render children()}
{/if}
