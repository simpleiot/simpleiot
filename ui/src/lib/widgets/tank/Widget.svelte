<script lang="ts">
	// A tank whose liquid follows the level binding, from the nautilus
	// faceplate; a temperature binding tints the liquid when present.
	import Tank from "#lib/vendor/nautilus/Tank.svelte"
	import type { WidgetProps } from "../registry.ts"

	let { label, values, width }: WidgetProps = $props()

	const level = $derived(values.level)
	const temp = $derived(values.temp)
	const dead = $derived(!level || level.state === "dead")
</script>

<div class="tank" class:dead>
	<Tank
		levelPct={dead ? 0 : (level?.value ?? 0)}
		tempC={temp?.state === "live" ? temp.value : 20}
		{label}
		width={width ?? 220}
	/>
</div>

<style>
	.tank {
		display: flex;
		justify-content: center;
	}

	.dead {
		opacity: 0.5;
	}
</style>
