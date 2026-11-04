<script lang="ts">
	// A radial gauge over a numeric range, from the nautilus faceplate.
	import Gauge from "#lib/vendor/nautilus/Gauge.svelte"
	import { multiplier, num, type WidgetProps } from "../registry.ts"

	let { label, values, options }: WidgetProps = $props()

	const v = $derived(values.value)
	const dead = $derived(!v || v.state === "dead")
</script>

<div class="gauge" class:dead>
	<Gauge
		value={dead ? num(options, "min", 0) : (v?.value ?? 0) * multiplier(options)}
		min={num(options, "min", 0)}
		max={num(options, "max", 100)}
		unit={options.unit ?? ""}
		{label}
		decimals={num(options, "decimals", 1)}
	/>
</div>

<style>
	.gauge {
		display: flex;
		justify-content: center;
		height: 100%;
	}

	.dead {
		opacity: 0.5;
	}
</style>
