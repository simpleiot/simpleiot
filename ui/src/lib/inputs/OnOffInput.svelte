<script lang="ts">
	// OnOffInput is a switch that sets setType. It pulses while the value
	// the device reports (type) differs from the one set.
	import { DataType, getValue } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"

	let {
		type,
		setType,
		key = "0",
		label,
	}: { type: string; setType: string; key?: string; label: string } = $props()

	const ctx = inputContext()

	const current = $derived(getValue(ctx.node.points, type, key))
	const set = $derived(getValue(ctx.node.points, setType, key))
	const on = $derived(set !== 0)

	function toggle() {
		ctx.onEditPoints([
			{
				type: setType,
				key,
				time: editTime(ctx),
				dataType: DataType.Float,
				value: on ? 0 : 1,
				text: "",
				tombstone: 0,
			},
		])
	}
</script>

<div class="onoff">
	<span class="label" style:width="{ctx.labelWidth}px">{label}:</span>
	<button
		type="button"
		role="switch"
		aria-checked={on}
		aria-label={label}
		class:on
		class:pending={current !== set}
		onclick={toggle}
	>
		<svg viewBox="0 0 48 24" width="100" aria-hidden="true">
			<rect class="track" x="0" y="0" width="48" height="24" rx="3" ry="3" />
			<rect class="knob" x={on ? 27 : 3} y="3" width="18" height="18" rx="3" ry="3" />
		</svg>
	</button>
</div>

<style>
	.onoff {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px 10px;
	}

	.label {
		text-align: right;
	}

	button {
		border: none;
		background: none;
		padding: 0;
		cursor: pointer;
		display: flex;
	}

	.track {
		fill: var(--switch-off);
	}

	.on .track {
		fill: var(--switch-on);
	}

	.pending .track {
		animation: pulse-off 2s infinite;
	}

	.pending.on .track {
		animation-name: pulse-on;
	}

	.knob {
		fill: #fff;
	}

	@keyframes pulse-off {
		50% {
			fill: var(--switch-off-fade);
		}
	}

	@keyframes pulse-on {
		50% {
			fill: var(--switch-on-fade);
		}
	}

	@media (max-width: 600px) {
		.label {
			text-align: left;
			width: auto !important;
		}
	}
</style>
