<script lang="ts">
	// A value tile: one number or text, large, with its unit.
	import { num, type WidgetProps } from "../registry.ts"

	let { label, values, options }: WidgetProps = $props()

	const v = $derived(values.value)
	const text = $derived.by(() => {
		if (!v || v.state === "dead") {
			return "—"
		}
		if (v.text !== "") {
			return v.text
		}
		return v.value.toFixed(num(options, "decimals", 1))
	})
</script>

<div class="value" class:dead={!v || v.state === "dead"} class:stale={v?.state === "stale"}>
	<span class="num">{text}</span>
	{#if options.unit}<span class="unit">{options.unit}</span>{/if}
	<span class="label">{label}</span>
</div>

<style>
	.value {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 4px 8px;
		height: 100%;
	}

	.num {
		font-size: 32px;
		font-weight: 650;
		font-variant-numeric: tabular-nums;
		line-height: 1.1;
	}

	.unit {
		font-size: 14px;
		color: var(--muted);
	}

	.label {
		flex-basis: 100%;
		font-size: 13px;
		color: var(--muted);
	}

	.stale .num {
		opacity: 0.6;
	}

	.dead .num {
		color: var(--muted);
	}
</style>
