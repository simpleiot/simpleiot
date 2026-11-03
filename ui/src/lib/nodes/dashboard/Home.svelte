<script lang="ts">
	// Home decides where "/" lands: the user's home dashboard when set, the
	// dashboard list when any dashboard is visible, else the node tree.
	import { untrack } from "svelte"
	import { goto } from "$app/navigation"
	import { resolve } from "$app/paths"
	import { auth } from "#lib/auth.svelte.ts"
	import { getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { store } from "#lib/siot/store.ts"
	import { dashboardType } from "./index.ts"

	$effect(() => {
		const user = auth.user
		if (user) {
			store.start(user.token, () => {
				auth.signOut()
			})
		}
	})

	let deciding = false
	async function decide() {
		if (deciding) {
			return
		}
		deciding = true
		const me = await store.resolveNode(store.userId)
		const home = me ? getText(me.points, PointType.HomeDashboard) : ""
		if (home !== "" && (await store.resolveNode(home))) {
			await goto(resolve("/dash/[id]", { id: home }), { replaceState: true })
			return
		}
		const any = await store.searchNodes("", 1, dashboardType)
		await goto(resolve(any.length > 0 ? "/dash" : "/nodes"), { replaceState: true })
	}

	$effect(() => {
		store.onConnected = () => void decide()
		if (store.ready) {
			untrack(() => void decide())
		}
	})
</script>

<p class="muted">Loading...</p>

<style>
	.muted {
		margin: 20px 16px;
		color: var(--muted);
	}
</style>
