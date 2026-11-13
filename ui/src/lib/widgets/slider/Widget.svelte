<script lang="ts">
	// A slider: the value over a range, and with an onWrite a setting. The
	// value is sent when the slider is let go, not while it moves, so a
	// drag is one write; confirm asks first. min, max, and step are in the
	// shown unit, and a multiplier scales the value for display and back
	// before it is sent, as on a value tile.
	import Confirm from "../Confirm.svelte"
	import { multiplier, num, wantsConfirm, type WidgetProps, type Write } from "../registry.ts"

	let {
		label,
		values,
		options,
		onWrite,
		control = false,
		editing = false,
	}: WidgetProps = $props()

	const v = $derived(values.value)
	const dead = $derived(!v || v.state === "dead")
	const live = $derived(v?.state === "live")
	const m = $derived(multiplier(options))
	const min = $derived(num(options, "min", 0))
	const max = $derived(num(options, "max", 100))
	const step = $derived(num(options, "step", 1))
	const decimals = $derived(num(options, "decimals", 0))
	const shown = $derived(live && v ? v.value * m : min)

	// draft is the position while the slider is held; null follows the point
	let draft = $state<number | null>(null)
	// pending is a write waiting to be confirmed
	let pending = $state<Write | null>(null)
	const position = $derived(draft ?? shown)

	const disabled = $derived(!onWrite || dead || editing)

	function send(n: number) {
		const w = { value: n / m, text: "" }
		if (wantsConfirm(options)) {
			pending = w
		} else {
			draft = null
			onWrite?.("value", w)
		}
	}
</script>

<div class="slider" class:dead class:stale={v?.state === "stale"}>
	<div class="top">
		<span class="num">{dead ? "—" : position.toFixed(decimals)}</span>
		{#if options.unit}<span class="unit">{options.unit}</span>{/if}
	</div>
	<input
		type="range"
		class:boxed={editing && control}
		{min}
		{max}
		{step}
		value={position}
		{disabled}
		aria-label="Set {label}"
		onclick={(e) => e.stopPropagation()}
		onpointerdown={(e) => e.stopPropagation()}
		oninput={(e) => (draft = Number(e.currentTarget.value))}
		onchange={(e) => send(Number(e.currentTarget.value))}
	/>
	<span class="label">{label}</span>
	{#if pending}
		<Confirm
			text="Set to {(pending.value * m).toFixed(decimals)}{options.unit ? ` ${options.unit}` : ''}?"
			onConfirm={() => {
				const w = pending
				pending = null
				draft = null
				if (w) {
					onWrite?.("value", w)
				}
			}}
			onCancel={() => {
				pending = null
				draft = null
			}}
		/>
	{/if}
</div>

<style>
	.slider {
		display: flex;
		flex-direction: column;
		gap: 4px;
		height: 100%;
	}

	.top {
		display: flex;
		align-items: baseline;
		gap: 8px;
	}

	.num {
		font-size: var(--num-size, 32px);
		font-weight: 650;
		font-variant-numeric: tabular-nums;
		line-height: 1.1;
	}

	.unit {
		font-size: calc(var(--num-size, 32px) * 0.45);
		color: var(--muted);
	}

	input {
		width: 100%;
		margin: 4px 0;
		cursor: pointer;
	}

	input:disabled {
		cursor: default;
	}

	.boxed {
		outline: 1px dashed var(--border);
		outline-offset: 4px;
	}

	.label {
		font-size: 15px;
		color: var(--muted);
	}

	.stale .num {
		opacity: 0.6;
	}

	.dead .num {
		color: var(--muted);
	}
</style>
