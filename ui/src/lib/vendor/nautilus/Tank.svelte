<script lang="ts">
	// Reactive tank graphic: liquid height follows level, liquid color follows
	// temperature, heater coil glows with heater power.
	import { tempColor } from "./color.ts"

	let {
		levelPct = 50,
		tempC = 20,
		heaterPct = 0,
		label = "",
		width = 220,
	}: {
		levelPct?: number
		tempC?: number
		heaterPct?: number
		label?: string
		width?: number
	} = $props()

	const X0 = 40,
		X1 = 180,
		Y0 = 26,
		Y1 = 216 // tank interior
	const innerH = Y1 - Y0

	let clamped = $derived(Math.max(0, Math.min(100, levelPct)))
	let liquidH = $derived((clamped / 100) * innerH)
	let liquidY = $derived(Y1 - liquidH)

	let liquid = $derived(tempColor(tempC))
	let liquidDeep = $derived(tempColor(tempC, 0.3))

	let heat = $derived(Math.max(0, Math.min(100, heaterPct)) / 100)
	// Quantized for animation timing: a continuously varying duration would
	// restart the bubble animation on every data frame.
	let heatQ = $derived(Math.max(0.25, Math.round(heat * 4) / 4))
	const uid = `tank${Math.random().toString(36).slice(2, 8)}`

	// Heater coil zigzag across the tank floor.
	const coil = (() => {
		let d = `M ${X0 + 14} ${Y1 - 16}`
		for (let i = 0; i < 6; i++) {
			const x = X0 + 14 + ((X1 - X0 - 28) / 6) * (i + 0.5)
			d += ` L ${x} ${Y1 - (i % 2 ? 16 : 30)}`
		}
		d += ` L ${X1 - 14} ${Y1 - 16}`
		return d
	})()
</script>

<svg
	viewBox="0 0 220 260"
	{width}
	role="img"
	aria-label={`Tank ${label}: ${clamped.toFixed(0)}%, ${tempC.toFixed(1)} degrees`}
>
	<defs>
		<linearGradient id="{uid}-liq" x1="0" y1="0" x2="0" y2="1">
			<stop class="tstop" offset="0" stop-color={liquid} />
			<stop class="tstop" offset="1" stop-color={liquidDeep} />
		</linearGradient>
		<clipPath id="{uid}-clip">
			<rect x={X0} y={Y0} width={X1 - X0} height={innerH} rx="10" />
		</clipPath>
		<!-- bubbles are clipped to the liquid, so they break at the surface
		     instead of rising into the empty vapor space above -->
		<clipPath id="{uid}-liqclip">
			<rect class="geo" x={X0} y={liquidY} width={X1 - X0} height={liquidH} />
		</clipPath>
		<!-- coil bbox is short, so the default ±10% filter region clips the
		     blur into hard edges — give it explicit headroom -->
		<filter id="{uid}-glow" x="-20%" y="-150%" width="140%" height="400%">
			<feGaussianBlur stdDeviation="3" />
		</filter>
	</defs>

	<!-- shell -->
	<rect
		x={X0 - 6}
		y={Y0 - 6}
		width={X1 - X0 + 12}
		height={innerH + 12}
		rx="14"
		fill="var(--surface-2, #232321)"
		stroke="var(--axis, #383835)"
		stroke-width="2"
	/>
	<rect x={X0} y={Y0} width={X1 - X0} height={innerH} rx="10" fill="var(--bg, #0d0d0d)" />

	<g clip-path="url(#{uid}-clip)">
		<rect class="geo" x={X0} y={liquidY} width={X1 - X0} height={liquidH} fill="url(#{uid}-liq)" />
		<!-- surface line -->
		{#if clamped > 1 && clamped < 99}
			<rect
				class="geo"
				x={X0}
				y={liquidY}
				width={X1 - X0}
				height="2"
				fill="rgba(255,255,255,0.35)"
			/>
		{/if}
		<!-- convection bubbles while heating -->
		{#if heat > 0.05}
			<g clip-path="url(#{uid}-liqclip)">
				{#each [0, 1, 2, 3, 4] as i (i)}
					<circle
						class="bubble"
						style="animation-delay: {i * 0.7}s; animation-duration: {3.4 - heatQ * 1.6}s"
						cx={X0 + 24 + i * 28}
						cy={Y1 - 20}
						r={2 + (i % 3)}
						fill="rgba(255,255,255,0.5)"
						opacity={0.25 + heat * 0.5}
					/>
				{/each}
			</g>
		{/if}
		<!-- heater coil -->
		{#if heat > 0.02}
			<path
				class="glowpath"
				d={coil}
				fill="none"
				stroke="#ff8c42"
				stroke-width="7"
				opacity={heat * 0.9}
				filter="url(#{uid}-glow)"
			/>
		{/if}
		<path
			class="coil"
			d={coil}
			fill="none"
			stroke={heat > 0.02 ? `hsl(${30 - heat * 22} 90% ${35 + heat * 25}%)` : "#4a4a47"}
			stroke-width="4"
			stroke-linecap="round"
		/>
	</g>

	<!-- level graduations -->
	{#each [0, 25, 50, 75, 100] as g (g)}
		<line
			x1={X1 + 8}
			x2={X1 + 16}
			y1={Y1 - (g / 100) * innerH}
			y2={Y1 - (g / 100) * innerH}
			stroke="var(--muted, #898781)"
			stroke-width="1"
		/>
		<text x={X1 + 20} y={Y1 - (g / 100) * innerH + 3.5} font-size="12" fill="var(--muted, #898781)"
			>{g}</text
		>
	{/each}

	{#if label}
		<text
			x="110"
			y="248"
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

	/* Data may arrive at 10 Hz; short linear transitions turn the steps into
	   continuous motion. */
	.geo {
		transition:
			y 0.18s linear,
			height 0.18s linear;
	}
	.tstop {
		transition: stop-color 0.5s linear;
	}
	.coil {
		transition: stroke 0.3s linear;
	}
	.glowpath {
		transition: opacity 0.3s linear;
	}
	.bubble {
		animation-name: rise;
		animation-timing-function: linear;
		animation-iteration-count: infinite;
		transition: opacity 0.3s linear;
	}
	@keyframes rise {
		from {
			transform: translateY(0);
			opacity: 0.6;
		}
		to {
			transform: translateY(-150px);
			opacity: 0;
		}
	}
</style>
