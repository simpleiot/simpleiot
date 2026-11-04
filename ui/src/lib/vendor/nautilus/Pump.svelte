<script lang="ts">
	// Centrifugal pump: impeller spins when running, spin rate follows speed.
	import { reducedMotion } from "./motion.ts"

	let {
		running = false,
		speedPct = 0,
		label = "",
		width = 120,
	}: { running?: boolean; speedPct?: number; label?: string; width?: number } = $props()

	// active = running fast enough to show as live (drives the color);
	// animate = actually spin/pulse (suppressed under reduced motion).
	let active = $derived(running && speedPct > 2)
	let animate = $derived(active && !reducedMotion())
	// 100% speed → 0.5s/rev; slower speed → proportionally slower. Speed is
	// quantized to 5% steps so feedback noise doesn't restart the animation.
	let speedQ = $derived(Math.max(5, Math.round(speedPct / 5) * 5))
	let period = $derived(animate ? 50 / speedQ : 0)
</script>

<svg
	viewBox="0 0 120 110"
	{width}
	role="img"
	aria-label={`Pump ${label}: ${running ? "running" : "stopped"}`}
>
	<!-- suction / discharge stubs -->
	<rect
		x="0"
		y="52"
		width="22"
		height="12"
		fill="var(--surface-2, #232321)"
		stroke="var(--axis, #383835)"
	/>
	<rect
		x="52"
		y="6"
		width="12"
		height="20"
		fill="var(--surface-2, #232321)"
		stroke="var(--axis, #383835)"
	/>

	<!-- volute -->
	<circle
		cx="58"
		cy="58"
		r="34"
		fill="var(--surface-2, #232321)"
		stroke="var(--axis, #383835)"
		stroke-width="3"
	/>
	<circle cx="58" cy="58" r="25" fill="var(--bg, #0d0d0d)" stroke="var(--axis, #383835)" />

	<!-- impeller -->
	<g class:spin={animate} style="--period: {period}s" transform-origin="58 58">
		{#each [0, 120, 240] as a (a)}
			<path
				d="M 58 58 Q 66 46 58 36"
				fill="none"
				stroke={active ? "var(--s1, #3987e5)" : "var(--muted, #898781)"}
				stroke-width="4"
				stroke-linecap="round"
				transform="rotate({a} 58 58)"
			/>
		{/each}
		<circle cx="58" cy="58" r="4" fill={active ? "var(--s1, #3987e5)" : "var(--muted, #898781)"} />
	</g>

	<!-- status lamp -->
	<circle
		cx="97"
		cy="24"
		r="6"
		fill={running ? "var(--good, #0ca30c)" : "var(--surface-2, #232321)"}
		stroke="var(--axis, #383835)"
	>
		{#if running && !reducedMotion()}<animate
				attributeName="opacity"
				values="1;0.6;1"
				dur="1.6s"
				repeatCount="indefinite"
			/>{/if}
	</circle>

	{#if label}
		<text
			x="58"
			y="104"
			text-anchor="middle"
			font-size="12"
			font-weight="600"
			fill="var(--ink-2, #c3c2b7)">{label}</text
		>
	{/if}
</svg>

<style>
	/* `width` stays a real attribute, but the rendered box also shrinks under
	   a narrower parent instead of overflowing it. */
	svg {
		max-width: 100%;
		height: auto;
	}
	.spin {
		animation: rot var(--period) linear infinite;
	}
	/* counterclockwise: the backward-curved blades must trail the rotation */
	@keyframes rot {
		to {
			transform: rotate(-360deg);
		}
	}
</style>
