<script lang="ts">
	// A status pill: on or off from a value, never color alone. With an
	// onWrite it is a switch: a click sends the other state, or asks first
	// when the confirm option is on.
	import Confirm from "../Confirm.svelte"
	import { wantsConfirm, type WidgetProps } from "../registry.ts"

	let { label, values, options, onWrite, control = false, editing = false }: WidgetProps = $props()

	// in edit mode a control wears a box, so it can be told from a display
	const boxed = $derived(editing && control)

	const v = $derived(values.value)
	const dead = $derived(!v || v.state === "dead")
	const on = $derived(!dead && (v?.value ?? 0) !== 0)
	const text = $derived(dead ? "no data" : on ? options.on : options.off)

	// pending is the state waiting to be confirmed
	let pending = $state<boolean | null>(null)

	function toggle(e: MouseEvent) {
		e.stopPropagation()
		if (wantsConfirm(options)) {
			pending = !on
		} else {
			onWrite?.("value", { value: on ? 0 : 1, text: "" })
		}
	}
</script>

<div class="status">
	{#if onWrite && !dead}
		<button
			class="pill switch"
			class:boxed
			class:on
			type="button"
			role="switch"
			aria-checked={on}
			aria-label="{label}: {text}"
			onclick={toggle}
		>
			<span class="icon" aria-hidden="true">{on ? "●" : "○"}</span>
			{text}
		</button>
	{:else}
		<span class="pill" class:on class:dead class:boxed>
			<span class="icon" aria-hidden="true">{on ? "●" : "○"}</span>
			{text}
		</span>
	{/if}
	<span class="label">{label}</span>
	{#if pending !== null}
		<Confirm
			text="Set {label} to {pending ? options.on : options.off}?"
			onConfirm={() => {
				const next = pending
				pending = null
				if (next !== null) {
					onWrite?.("value", { value: next ? 1 : 0, text: "" })
				}
			}}
			onCancel={() => (pending = null)}
		/>
	{/if}
</div>

<style>
	.status {
		display: flex;
		flex-direction: column;
		gap: 6px;
		height: 100%;
	}

	/* an outlined pill with no fill: the outline is the status, the dot
	   its color; a switch answers the pointer with a blue outline, and in
	   edit mode a control is filled so it can be told from a display */
	.pill {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		align-self: flex-start;
		max-width: 100%;
		box-sizing: border-box;
		margin: 0;
		padding: 4px 14px;
		white-space: nowrap;
		border: 1px solid var(--border);
		border-radius: 999px;
		font: inherit;
		font-size: var(--pill-size, 18px);
		font-weight: 600;
		color: inherit;
		background: none;
		--c: var(--muted);
	}

	.pill.on {
		--c: var(--green);
		border-color: var(--green);
	}

	.switch {
		cursor: pointer;
	}

	.switch:hover,
	.switch:focus-visible {
		border-color: var(--blue);
	}

	.boxed {
		background: var(--detail-bg);
	}

	.icon {
		color: var(--c);
	}

	.dead {
		color: var(--muted);
	}

	.label {
		font-size: 13px;
		color: var(--muted);
	}
</style>
