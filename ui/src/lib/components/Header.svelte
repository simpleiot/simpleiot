<script lang="ts">
	import { asset, resolve } from "$app/paths"
	import Moon from "@lucide/svelte/icons/moon"
	import Sun from "@lucide/svelte/icons/sun"
	import { prefs } from "#lib/prefs.svelte.ts"
	import { effectiveTheme, toggleTheme } from "#lib/theme.svelte.ts"
	import type { Snippet } from "svelte"

	let {
		email,
		badge = "",
		onSignOut,
		search,
	}: {
		email: string
		badge?: string
		onSignOut: () => void
		search?: Snippet
	} = $props()

	// re-read when the preference changes
	const dark = $derived(
		prefs.theme !== undefined ? prefs.theme === "dark" : effectiveTheme() === "dark"
	)
</script>

<header>
	<a href={resolve("/")} class="logo"
		><img src={asset("siot-logo.svg")} alt="Simple IoT" height="32" /></a
	>
	{#if badge}
		<span class="badge" role="status">{badge}</span>
	{/if}
	<div class="spacer"></div>
	{#if search}
		<div class="search">{@render search()}</div>
	{/if}
	<button
		class="icon"
		onclick={toggleTheme}
		title={dark ? "Light theme" : "Dark theme"}
		aria-label={dark ? "Switch to the light theme" : "Switch to the dark theme"}
	>
		{#if dark}<Sun size={20} />{:else}<Moon size={20} />{/if}
	</button>
	<button class="btn blue" onclick={onSignOut}>sign out {email}</button>
</header>

<style>
	header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px 20px;
	}

	.logo {
		display: flex;
	}

	.spacer {
		flex: 1;
	}

	.icon {
		display: flex;
		padding: 6px;
		border: none;
		border-radius: 50%;
		background: none;
		cursor: pointer;
	}

	.icon:hover {
		background: var(--hover);
	}

	/* on a phone the search takes its own line, under the buttons */
	@media (max-width: 600px) {
		.search {
			order: 1;
			flex-basis: 100%;
		}
	}
</style>
