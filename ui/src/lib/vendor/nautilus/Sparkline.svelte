<script lang="ts">
	// Minimal single-series line for stat tiles and diagnostics rows.
	//
	// Two modes. Static (default): the line is fit to the box and redrawn in
	// place. Scrolling (pass `endIndex`, the absolute sample index of the
	// newest value, e.g. a scan counter): samples are plotted at absolute
	// positions and a transitioned transform slides the window left as new
	// samples arrive, so live series scroll smoothly instead of snapping.
	//
	// `yMin`/`yMax` fix the vertical scale so it does not jump as data streams
	// in; omit either to auto-fit that side to the series' own min/max. An
	// empty series and a one-sample series both render safely — no NaN path,
	// and one sample draws as a dot rather than nothing.
	import { sparklineGeometry, sparklineMetrics } from "./sparkline.ts"

	let {
		values,
		color = "var(--s1)",
		height = 44,
		yMin,
		yMax,
		endIndex,
		windowSize,
		compact = false,
	}: {
		values: number[]
		color?: string
		height?: number
		yMin?: number
		yMax?: number
		/** Absolute index of values[values.length-1]; enables scrolling mode. */
		endIndex?: number
		/** Samples spanning the full width in scrolling mode (default: values.length). */
		windowSize?: number
		/** Tighter insets and a thinner stroke, for dense card/list contexts
		 *  (e.g. `EquipmentCard`'s sparkline slot) rather than a stat tile. */
		compact?: boolean
	} = $props()

	let w = $state(240)

	let metrics = $derived(sparklineMetrics(compact))
	let scrolling = $derived(endIndex !== undefined)
	let geometry = $derived(
		sparklineGeometry(values, { width: w, height, yMin, yMax, compact, endIndex, windowSize })
	)
</script>

<div bind:clientWidth={w} style="width: 100%">
	<svg
		viewBox="0 0 {w} {height}"
		style="width: 100%; display: block"
		role="img"
		aria-label="sparkline"
	>
		{#if values.length > 1}
			<g class:scroll={scrolling} style="transform: translateX({geometry.tx.toFixed(2)}px)">
				<path
					d={geometry.path}
					fill="none"
					stroke={color}
					stroke-width={metrics.strokeWidth}
					stroke-linejoin="round"
				/>
			</g>
		{:else if geometry.singlePoint}
			<g class:scroll={scrolling} style="transform: translateX({geometry.tx.toFixed(2)}px)">
				<circle
					cx={geometry.singlePoint.x.toFixed(2)}
					cy={geometry.singlePoint.y.toFixed(2)}
					r={metrics.strokeWidth + 1}
					fill={color}
				/>
			</g>
		{/if}
	</svg>
</div>

<style>
	g.scroll {
		transition: transform 0.3s linear;
	}
	@media (prefers-reduced-motion: reduce) {
		g.scroll {
			transition: none;
		}
	}
</style>
