<script lang="ts">
	import { asset } from "$app/paths"
	import { auth } from "#lib/auth.svelte.ts"
	import { signIn } from "#lib/siot/http.ts"

	let email = $state("")
	let password = $state("")
	let error = $state("")
	let busy = $state(false)

	async function submit(e: SubmitEvent) {
		e.preventDefault()
		if (email === "" || busy) {
			return
		}
		busy = true
		try {
			const user = await signIn(email, password)
			if (user) {
				error = ""
				auth.signIn(user)
			} else {
				error = "Invalid login"
			}
		} catch {
			error = "Login Failure"
		} finally {
			busy = false
		}
	}
</script>

<svelte:head>
	<title>SIOT Sign In</title>
</svelte:head>

<main>
	<form onsubmit={submit}>
		<img src={asset("siot-logo.svg")} alt="Simple IoT" height="40" />
		{#if error}
			<div class="error" role="alert">{error}</div>
		{/if}
		<h2>Sign in</h2>
		<label>
			Email
			<input
				type="text"
				inputmode="email"
				autocomplete="username"
				placeholder="email"
				value={email}
				oninput={(e) => (email = e.currentTarget.value.toLowerCase())}
			/>
		</label>
		<label>
			Password
			<input
				type="password"
				autocomplete="current-password"
				placeholder="password"
				bind:value={password}
			/>
		</label>
		<button class="btn" class:blue={email !== ""} class:gray={email === ""} disabled={busy}>
			Sign In
		</button>
	</form>
</main>

<style>
	main {
		min-height: 100vh;
		display: grid;
		place-items: center;
		padding: 16px;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: 16px;
		width: min(100%, 320px);
	}

	img {
		align-self: center;
		margin-bottom: 16px;
	}

	h2 {
		margin-top: 16px;
	}

	label {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	button {
		align-self: flex-end;
	}

	.error {
		padding: 8px 16px;
		font-size: 14px;
		font-weight: 600;
		color: #fff;
		background: var(--error-bg);
		border-radius: 4px;
		text-align: center;
	}
</style>
