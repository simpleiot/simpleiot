<script lang="ts">
	// A value tile: one number or text, large, with its unit. With an
	// onWrite it is a setting: a click on the value opens an edit box, and
	// Enter sends the new value, or asks first when the confirm option is
	// on. Until it is edited the value looks like any other.
	import Confirm from "../Confirm.svelte"
	import { num, wantsConfirm, type WidgetProps, type Write } from "../registry.ts"

	let {
		label,
		values,
		options,
		onWrite,
		control = false,
		editing: dashEditing = false,
	}: WidgetProps = $props()

	// in the dashboard's edit mode a control wears a box, so it can be told
	// from a display
	const boxed = $derived(dashEditing && control)

	const v = $derived(values.value)
	const dead = $derived(!v || v.state === "dead")
	const isText = $derived(v !== undefined && v.text !== "")
	const decimals = $derived(num(options, "decimals", 1))
	const text = $derived.by(() => {
		if (!v || dead) {
			return "—"
		}
		if (v.text !== "") {
			return v.text
		}
		return v.value.toFixed(decimals)
	})

	let editing = $state(false)
	// a number input binds a number, a text input a string
	let draft = $state<string | number>("")
	let box: HTMLInputElement | undefined = $state()
	// pending is a write waiting to be confirmed
	let pending = $state<Write | null>(null)

	function begin(e: MouseEvent) {
		e.stopPropagation()
		if (!onWrite || dead || !v) {
			return
		}
		draft = isText ? v.text : String(v.value)
		editing = true
		requestAnimationFrame(() => box?.select())
	}

	function commit() {
		editing = false
		if (!onWrite) {
			return
		}
		const t = String(draft).trim()
		let w: Write
		if (isText) {
			w = { value: 0, text: t }
		} else {
			const n = Number(t)
			if (t === "" || !Number.isFinite(n)) {
				return
			}
			w = { value: n, text: "" }
		}
		if (wantsConfirm(options)) {
			pending = w
		} else {
			onWrite("value", w)
		}
	}

	function onkeydown(e: KeyboardEvent) {
		e.stopPropagation()
		if (e.key === "Enter") {
			commit()
		} else if (e.key === "Escape") {
			editing = false
		}
	}

	const pendingText = $derived(
		pending ? (pending.text !== "" ? pending.text : pending.value.toFixed(decimals)) : ""
	)
</script>

<div class="value" class:dead class:stale={v?.state === "stale"}>
	{#if editing}
		<input
			class="box"
			type={isText ? "text" : "number"}
			step="any"
			bind:value={draft}
			bind:this={box}
			aria-label="Set {label}"
			onclick={(e) => e.stopPropagation()}
			{onkeydown}
			onblur={() => (editing = false)}
		/>
	{:else if onWrite && !dead}
		<button class="num settable" class:boxed type="button" aria-label="Edit {label}" onclick={begin}
			>{text}</button
		>
	{:else}
		<span class="num" class:boxed>{text}</span>
	{/if}
	{#if options.unit}<span class="unit">{options.unit}</span>{/if}
	<span class="label">{label}</span>
	{#if pending}
		<Confirm
			text="Set to {pendingText}{options.unit ? ` ${options.unit}` : ''}?"
			onConfirm={() => {
				const w = pending
				pending = null
				if (w) {
					onWrite?.("value", w)
				}
			}}
			onCancel={() => (pending = null)}
		/>
	{/if}
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
		font-size: var(--num-size, 32px);
		font-weight: 650;
		font-variant-numeric: tabular-nums;
		line-height: 1.1;
	}

	/* a settable value looks like any other until it is pointed at */
	.settable {
		padding: 0;
		margin: 0;
		border: none;
		background: none;
		font: inherit;
		font-size: var(--num-size, 32px);
		font-weight: 650;
		color: inherit;
		cursor: text;
	}

	.settable:hover,
	.settable:focus-visible {
		text-decoration: underline dotted var(--muted);
		text-underline-offset: 6px;
	}

	.boxed {
		padding: 0 6px;
		border: 1px dashed var(--border);
		border-radius: 6px;
		background: var(--hover);
	}

	.box {
		width: min(12em, 100%);
		font-size: calc(var(--num-size, 32px) * 0.75);
		font-weight: 600;
	}

	.unit {
		font-size: calc(var(--num-size, 32px) * 0.45);
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
