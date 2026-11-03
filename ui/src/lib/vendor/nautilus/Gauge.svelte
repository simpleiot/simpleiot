<script lang="ts">
	// Radial gauge. A 240° arc by default; `sweep` (or `gap`) opens it out to a
	// full ring for the donut style legacy SCADA packages draw.
	let {
		value = 0,
		min = 0,
		max = 100,
		unit = "",
		label = "",
		color = "var(--s1)",
		setpoint,
		decimals = 1,
		width = 170,
		sweep = 240,
		gap,
		thickness = 10,
	}: {
		value?: number
		min?: number
		max?: number
		unit?: string
		label?: string
		color?: string
		/** Draws a marker ring at this value; omit for none. */
		setpoint?: number
		decimals?: number
		width?: number
		/**
		 * Degrees the scale spans, centred on twelve o'clock. 240 (the default)
		 * is the classic three-quarter dial; 360 is a closed ring. Clamped to
		 * 30…360. Ignored when `gap` is given.
		 */
		sweep?: number
		/**
		 * The complement of `sweep`, for the donut idiom: degrees of ring left
		 * OPEN at the bottom. `gap={30}` is a 330° ring. Takes precedence over
		 * `sweep` when set.
		 */
		gap?: number
		/** Arc stroke width, in the gauge's own 170-unit coordinate space. */
		thickness?: number
	} = $props()

	const CX = 85,
		CY = 82,
		R = 60

	const span = $derived(Math.max(30, Math.min(360, gap === undefined ? sweep : 360 - gap)))
	/** A closed (or near-closed) ring needs its end labels moved off the seam. */
	const closed = $derived(span > 300)

	// The drawing box grows downward as the arc closes: at 240° the ends sit at
	// y=112, at 360° they meet at y=142. +18 leaves room for the end labels and
	// keeps the default exactly the historical 170×130 box.
	const height = $derived(Math.ceil(CY - R * Math.cos((span / 2) * (Math.PI / 180))) + 18)

	function polar(frac: number): [number, number] {
		const a = ((-span / 2 + span * frac) * Math.PI) / 180
		return [CX + R * Math.sin(a), CY - R * Math.cos(a)]
	}

	function arc(f0: number, f1: number) {
		const [x0, y0] = polar(f0)
		// A literal 360° arc collapses to a point in SVG — back off a hair so the
		// ring closes visually instead of vanishing.
		const [x1, y1] = polar(span >= 360 && f1 >= 1 ? 0.9995 : f1)
		const large = (f1 - f0) * span > 180 ? 1 : 0
		return `M ${x0} ${y0} A ${R} ${R} 0 ${large} 1 ${x1} ${y1}`
	}

	let frac = $derived(Math.max(0, Math.min(1, (value - min) / (max - min || 1))))
	let spFrac = $derived(
		setpoint === undefined ? null : Math.max(0, Math.min(1, (setpoint - min) / (max - min || 1)))
	)
</script>

<svg
	viewBox={`0 0 170 ${height}`}
	{width}
	role="img"
	aria-label={`${label}: ${value.toFixed(decimals)} ${unit}`}
>
	<path
		d={arc(0, 1)}
		fill="none"
		stroke="var(--grid)"
		stroke-width={thickness}
		stroke-linecap="round"
	/>
	{#if frac > 0.005}
		<path
			class="val"
			d={arc(0, frac)}
			fill="none"
			stroke={color}
			stroke-width={thickness}
			stroke-linecap="round"
		/>
	{/if}
	{#if spFrac !== null}
		{@const [sx, sy] = polar(spFrac)}
		<circle
			class="sp"
			cx={sx}
			cy={sy}
			r="5"
			fill="var(--surface)"
			stroke="var(--s2)"
			stroke-width="2.5"
		>
			<title>Setpoint: {setpoint}</title>
		</circle>
	{/if}
	<text
		x={CX}
		y={CY + 2}
		text-anchor="middle"
		font-size="26"
		font-weight="650"
		fill="var(--ink)"
		class="num"
	>
		{value.toFixed(decimals)}
	</text>
	<text x={CX} y={CY + 18} text-anchor="middle" font-size="12" fill="var(--muted)">{unit}</text>
	<text x={CX} y={CY + 42} text-anchor="middle" font-size="12" font-weight="600" fill="var(--ink-2)"
		>{label}</text
	>
	{#each [0, 1] as f (f)}
		{@const [tx, ty] = polar(f)}
		<text
			x={closed ? tx + (f ? 8 : -8) : tx}
			y={closed ? ty + 4 : ty + 16}
			text-anchor={closed ? (f ? "start" : "end") : "middle"}
			font-size="12"
			fill="var(--muted)"
			class="num">{f ? max : min}</text
		>
	{/each}
</svg>

<style>
	/* `width` stays a real attribute (it drives the gauge's own coordinate
	   math via `thickness`/box sizing), but the rendered box also shrinks
	   under a narrower parent instead of overflowing it. */
	svg {
		max-width: 100%;
		height: auto;
	}

	/* Smooth updates. `d` interpolates in Chromium when the path structure is
	   unchanged; elsewhere it falls back to stepping. */
	.val {
		transition: d 0.18s linear;
	}
	.sp {
		transition:
			cx 0.18s linear,
			cy 0.18s linear;
	}
</style>
