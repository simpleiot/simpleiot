<script lang="ts">
	// A valve whose disc turns with the open binding, from the nautilus
	// faceplate.
	import Valve from "#lib/vendor/nautilus/Valve.svelte"
	import type { WidgetProps } from "../registry.ts"

	let { label, values, width }: WidgetProps = $props()

	const open = $derived(values.open)
	const dead = $derived(!open || open.state === "dead")
</script>

<div class="valve" class:dead>
	<Valve openPct={dead ? 0 : (open?.value ?? 0)} {label} width={width ?? 90} />
</div>

<style>
	.valve {
		display: flex;
		justify-content: center;
	}

	.dead {
		opacity: 0.5;
	}
</style>
