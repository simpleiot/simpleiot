<script lang="ts">
	// One dashboard, by node ID.
	import { page } from "$app/state"
	import DashView from "#lib/nodes/dashboard/DashView.svelte"
</script>

<DashView id={page.params.id ?? ""} />
