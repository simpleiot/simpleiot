<script lang="ts">
	// A status pill: on or off from a value, never color alone. With an
	// onWrite it is a switch: a click sends the other state, or asks first
	// when the confirm option is on.
	import Confirm from "../Confirm.svelte"
	import { wantsConfirm, type WidgetProps } from "../registry.ts"

	let { label, values, options, onWrite }: WidgetProps = $props()

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
		<span class="pill" class:on class:dead>
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

	.pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		align-self: flex-start;
		padding: 4px 12px;
		border: 1px solid var(--border);
		border-radius: 999px;
		font: inherit;
		font-weight: 550;
		color: inherit;
		background: var(--detail-bg);
		--c: var(--muted);
	}

	.pill.on {
		--c: var(--green);
	}

	.switch {
		cursor: pointer;
	}

	.switch:hover {
		background: var(--hover);
	}

	.switch.on {
		border-color: var(--green);
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
