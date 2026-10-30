<script lang="ts">
	import SvgIcon from "./SvgIcon.svelte"

	let { size = 24 }: { size?: number | string } = $props()
</script>

<SvgIcon {size}>
	<g transform="scale(1.0,1.5) translate(0,-4)" stroke-width="1.5">
		<path d="m 1,9 h 22.428778 l -3.364317,5.685801 H 4.3643165 Z" />
		<ellipse stroke-width="0.8" cx="5.48" cy="10.75" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="8.85" cy="10.75" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="12.21" cy="10.75" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="15.58" cy="10.75" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="18.94" cy="10.75" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="7.16" cy="12.41" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="10.53" cy="12.41" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="13.90" cy="12.41" rx="0.56" ry="0.57" />
		<ellipse stroke-width="0.8" cx="17.26" cy="12.41" rx="0.56" ry="0.57" />
	</g>
</SvgIcon>
