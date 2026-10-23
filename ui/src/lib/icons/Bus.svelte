<script lang="ts">
	import SvgIcon from "./SvgIcon.svelte"

	let { size = 24 }: { size?: number | string } = $props()
</script>

<SvgIcon {size}>
	<line x1="11" y1="3" x2="11" y2="14" />
	<polyline points="3 14 3 9 19 9 19 14" />
	<rect fill="currentColor" stroke="none" x="0" y="14" width="6" height="5" />
	<rect fill="currentColor" stroke="none" x="8" y="14" width="6" height="5" />
	<rect fill="currentColor" stroke="none" x="16" y="14" width="6" height="5" />
</SvgIcon>
