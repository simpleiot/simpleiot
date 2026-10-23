<script lang="ts">
	import SvgIcon from "./SvgIcon.svelte"

	let { size = 24 }: { size?: number | string } = $props()
</script>

<SvgIcon {size}>
	<g stroke-width="0" fill="currentColor">
		<polyline points="3 11 11 11 11 7 13 7 13 11 21 11 12 2 3 11" />
		<polyline points="3 13 11 13 11 17 13 17 13 13 21 13 12 22 3 13" />
	</g>
</SvgIcon>
