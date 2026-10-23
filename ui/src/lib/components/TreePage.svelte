<script lang="ts">
	// TreePage is the node tree with its header. Both "/" and
	// "/nodes/[id]" render it; the store outlives either.
	import ClipboardIcon from "@lucide/svelte/icons/clipboard"
	import X from "@lucide/svelte/icons/x"
	import { untrack } from "svelte"
	import { beforeNavigate, goto } from "$app/navigation"
	import { resolve } from "$app/paths"
	import { auth } from "#lib/auth.svelte.ts"
	import { store } from "#lib/siot/store.ts"
	import { confirmDiscard } from "#lib/unsaved.ts"
	import Header from "./Header.svelte"
	import IconButton from "./IconButton.svelte"
	import NodeTree from "./NodeTree.svelte"

	let { deepLink }: { deepLink?: string } = $props()

	let notFound = $state(false)

	$effect(() => {
		const user = auth.user
		if (user) {
			store.start(user.token, () => {
				auth.signOut()
			})
		}
	})

	// the URL follows the open detail
	store.onDetail = (node, open) => {
		if (open) {
			void goto(resolve("/nodes/[id]", { id: node.id }), {
				shallow: true,
				replace: true,
			})
		} else if (location.pathname.endsWith(`/nodes/${encodeURIComponent(node.id)}`)) {
			void goto(resolve("/"), { shallow: true, replace: true })
		}
	}

	// a deep link opens once the tree is there
	let revealed = ""
	$effect(() => {
		const id = deepLink
		store.onConnected = () => {
			if (id && revealed !== id) {
				revealed = id
				void store.reveal(id).then((key) => {
					notFound = key === null
					if (key) {
						requestAnimationFrame(() => {
							document
								.querySelector(`[data-key="${CSS.escape(key)}"]`)
								?.scrollIntoView({ block: "start", behavior: "smooth" })
						})
					}
				})
			}
		}
		if (store.ready) {
			untrack(() => store.onConnected?.())
		}
	})

	function signOut() {
		if (!confirmDiscard()) {
			return
		}
		store.discardEdits()
		void store.stop()
		auth.signOut()
	}

	beforeNavigate(({ cancel, type }) => {
		if (type !== "leave" && !confirmDiscard()) {
			cancel()
		}
	})

	function onbeforeunload(e: BeforeUnloadEvent) {
		if (store.hasEdits) {
			e.preventDefault()
		}
	}
</script>

<svelte:window {onbeforeunload} />

<svelte:head>
	<title>SIOT</title>
</svelte:head>

<div class="page">
	<Header
		email={auth.user?.email ?? ""}
		badge={store.connected ? "" : "connecting..."}
		onSignOut={signOut}
	>
		{#snippet search()}
			<input
				class="search"
				type="search"
				placeholder={store.searching ? "loading the tree..." : "search"}
				aria-label="Search nodes by description, type, or ID"
				value={store.query}
				oninput={(e) => void store.setQuery(e.currentTarget.value)}
			/>
		{/snippet}
	</Header>
	{#if store.error}
		<div class="error" role="alert">
			<span>{store.error}</span>
			<IconButton label="Dismiss" onclick={() => store.dismissError()}><X size={18} /></IconButton>
		</div>
	{/if}
	<main>
		<div class="title">
			<h2>Nodes</h2>
			{#if store.clipboard}
				<span class="clip">
					<ClipboardIcon size={20} />
					<em>{store.clipboard.description}</em>
					<small>({store.clipboard.id})</small>
					<IconButton label="Clear the copied node" onclick={() => store.clearClipboard()}>
						<X size={20} />
					</IconButton>
				</span>
			{/if}
		</div>
		{#if notFound}
			<p class="not-found" role="status">Node {deepLink} was not found.</p>
		{/if}
		<NodeTree />
	</main>
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

	main {
		display: flex;
		flex-direction: column;
		gap: 32px;
	}

	.title {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}

	.clip {
		display: inline-flex;
		align-items: center;
		gap: 10px;
	}

	small {
		font-size: 12px;
	}

	.error {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 4px 16px;
		font-size: 14px;
		font-weight: 600;
		color: #fff;
		background: var(--error-bg);
		border-radius: 4px;
	}

	.search {
		width: min(16em, 100%);
	}

	.not-found {
		margin: 0;
		color: var(--muted);
	}
</style>
