<script lang="ts">
	// Control valve symbol; the stem indicator rotates with position
	// (0% = closed/perpendicular, 100% = open/inline).
	let {
		openPct = 0,
		label = "",
		width = 90,
	}: { openPct?: number; label?: string; width?: number } = $props()

	let open = $derived(Math.max(0, Math.min(100, openPct)))
	let angle = $derived(90 - (open / 100) * 90)
</script>

<svg viewBox="0 0 90 80" {width} role="img" aria-label={`Valve ${label}: ${open.toFixed(0)}% open`}>
	<!-- body: two triangles -->
	<path
		d="M 12 28 L 45 44 L 12 60 Z"
		fill="var(--surface-2, #232321)"
		stroke="var(--axis, #383835)"
		stroke-width="2"
	/>
	<path
		d="M 78 28 L 45 44 L 78 60 Z"
		fill="var(--surface-2, #232321)"
		stroke="var(--axis, #383835)"
		stroke-width="2"
	/>
	<!-- actuator -->
	<line x1="45" y1="44" x2="45" y2="22" stroke="var(--axis, #383835)" stroke-width="3" />
	<rect
		x="31"
		y="8"
		width="28"
		height="14"
		rx="3"
		fill="var(--surface-2, #232321)"
		stroke="var(--axis, #383835)"
		stroke-width="2"
	/>
	<!-- disc position indicator -->
	<g transform="translate(45 44)">
		<line
			class="disc"
			x1="-9"
			y1="0"
			x2="9"
			y2="0"
			style="transform: rotate({angle}deg)"
			stroke={open > 2 ? "var(--s3, #199e70)" : "var(--serious, #ec835a)"}
			stroke-width="4"
			stroke-linecap="round"
		/>
	</g>
	<text
		x="45"
		y="76"
		text-anchor="middle"
		font-size="12"
		font-weight="600"
		fill="var(--ink-2, #c3c2b7)"
	>
		{label ? `${label} · ` : ""}{open.toFixed(0)}%
	</text>
</svg>

<style>
	/* `width` stays a real attribute, but the rendered box also shrinks under
	   a narrower parent instead of overflowing it. */
	svg {
		max-width: 100%;
		height: auto;
	}
	.disc {
		transition:
			transform 0.25s linear,
			stroke 0.25s linear;
	}
</style>
