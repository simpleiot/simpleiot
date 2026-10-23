<script lang="ts">
	import { auth } from "#lib/auth.svelte.ts"
	import Header from "#lib/components/Header.svelte"
</script>

<svelte:head>
	<title>SIOT</title>
</svelte:head>

<div class="page">
	<Header email={auth.user?.email ?? ""} onSignOut={() => auth.signOut()} />
	<p>Signed in as {auth.user?.email}.</p>
</div>

<style>
	.page {
		max-width: 1280px;
		margin: 0 auto;
		padding: 20px 16px;
		display: flex;
		flex-direction: column;
		gap: 32px;
	}
</style>
