<script lang="ts">
	import SvgIcon from "./SvgIcon.svelte"

	let { size = 24 }: { size?: number | string } = $props()
</script>

<SvgIcon {size}>
	<polyline points="3 6 3 16" />
	<polyline points="12 3 8 19" />
	<ellipse cx="18" cy="11" rx="3" ry="5" />
</SvgIcon>
