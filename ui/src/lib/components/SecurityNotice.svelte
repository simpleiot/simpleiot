<script lang="ts">
	import X from "@lucide/svelte/icons/x"
	import type { User } from "#lib/auth.svelte.ts"
	import IconButton from "./IconButton.svelte"

	// What the server reported at sign-in about defaults that suit a bench
	// or a closed network and should change before the instance is reachable
	// from anywhere else. Shown after sign-in only, never on the sign-in
	// page, and cleared by the next sign-in once the default is changed.
	// Dismissing it hides it until the page is reloaded; the layout keeps
	// it mounted across navigation.
	let { user }: { user: User } = $props()

	let dismissed = $state(false)
</script>

{#if !dismissed && (user.defaultPassword || user.noAuthToken)}
	<div class="notice" role="status">
		<div class="text">
			{#if user.defaultPassword}
				<p>
					The <strong>{user.email}</strong> account still has the default password. Change it in that
					user's settings before this instance is reachable from other networks.
				</p>
			{/if}
			{#if user.noAuthToken}
				<p>
					No auth token is set (<code>SIOT_AUTH_TOKEN</code>), so this instance accepts NATS and
					MQTT connections without credentials. Set one before this instance is reachable from other
					networks.
				</p>
			{/if}
		</div>
		<IconButton label="Dismiss" onclick={() => (dismissed = true)}><X size={18} /></IconButton>
	</div>
{/if}

<style>
	.notice {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		font-size: 14px;
		color: #000;
		background: var(--highlight-bg);
		border-bottom: 1px solid var(--orange);
	}

	.text {
		flex: 1;
	}

	p {
		margin: 4px 0;
	}
</style>
