<script lang="ts">
	// One dashboard in edit mode with one widget selected, by node IDs.
	import { page } from "$app/state"
	import DashView from "#lib/nodes/dashboard/DashView.svelte"
</script>

<DashView id={page.params.id ?? ""} select={page.params.widget ?? ""} />
