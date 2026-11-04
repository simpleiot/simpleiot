<script lang="ts">
	// A pump whose impeller turns while the running binding is on, at the
	// speed binding's rate, from the nautilus faceplate.
	import Pump from "#lib/vendor/nautilus/Pump.svelte"
	import type { WidgetProps } from "../registry.ts"

	let { label, values, width }: WidgetProps = $props()

	const running = $derived(values.running)
	const speed = $derived(values.speed)
	const dead = $derived(!running || running.state === "dead")
</script>

<div class="pump" class:dead>
	<Pump
		running={!dead && (running?.value ?? 0) !== 0}
		speedPct={speed?.state === "live" ? speed.value : 100}
		{label}
		width={width ?? 120}
	/>
</div>

<style>
	.pump {
		display: flex;
		justify-content: center;
	}

	.dead {
		opacity: 0.5;
	}
</style>
