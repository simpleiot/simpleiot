<script lang="ts">
	// One dashboard by node ID, and with a widget ID, in edit mode with that
	// widget selected. One route for both, so selecting a widget changes the
	// address without remounting the page.
	import { page } from "$app/state"
	import DashView from "#lib/nodes/dashboard/DashView.svelte"
</script>

<DashView id={page.params.id ?? ""} select={page.params.widget ?? ""} />
