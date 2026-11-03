<script lang="ts">
	// A live trend: the last values of one point as they arrive, on the
	// nautilus sparkline. The window is a count of samples kept in the
	// browser; history from the store comes later.
	import Sparkline from "#lib/vendor/nautilus/Sparkline.svelte"
	import { num, type WidgetProps } from "../registry.ts"

	let { label, values, options }: WidgetProps = $props()

	const v = $derived(values.value)
	const dead = $derived(!v || v.state === "dead")
	const window = $derived(Math.max(2, Math.floor(num(options, "window", 120))))

	// samples are appended when the point's time changes
	let samples = $state<number[]>([])
	let lastTime = ""
	$effect(() => {
		const p = v
		if (!p || p.state !== "live" || p.time === lastTime) {
			return
		}
		lastTime = p.time
		samples = [...samples, p.value].slice(-window)
	})

	const minOpt = $derived(
		options.min === "" || options.min === undefined ? undefined : num(options, "min", 0)
	)
	const maxOpt = $derived(
		options.max === "" || options.max === undefined ? undefined : num(options, "max", 0)
	)
	const latest = $derived(dead || !v ? "—" : v.value.toFixed(num(options, "decimals", 1)))
</script>

<div class="trend" class:dead>
	<div class="head">
		<span class="label">{label}</span>
		<span class="num"
			>{latest}{#if options.unit}<span class="unit"> {options.unit}</span>{/if}</span
		>
	</div>
	<div class="chart">
		<Sparkline values={samples} height={60} yMin={minOpt} yMax={maxOpt} />
	</div>
</div>

<style>
	.trend {
		display: flex;
		flex-direction: column;
		gap: 6px;
		height: 100%;
	}

	.head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 8px;
	}

	.label {
		font-size: 13px;
		color: var(--muted);
	}

	.num {
		font-weight: 650;
		font-variant-numeric: tabular-nums;
	}

	.unit {
		font-weight: 400;
		color: var(--muted);
	}

	.chart {
		flex: 1;
		min-height: 60px;
	}

	.dead {
		opacity: 0.5;
	}
</style>
