<script lang="ts">
	// A value tile: one number or text, large, with its unit. With an
	// onWrite it is a setting: a click on the value opens an edit box, and
	// Enter sends the new value.
	import { num, type WidgetProps } from "../registry.ts"

	let { label, values, options, onWrite }: WidgetProps = $props()

	const v = $derived(values.value)
	const dead = $derived(!v || v.state === "dead")
	const isText = $derived(v !== undefined && v.text !== "")
	const text = $derived.by(() => {
		if (!v || dead) {
			return "—"
		}
		if (v.text !== "") {
			return v.text
		}
		return v.value.toFixed(num(options, "decimals", 1))
	})

	let editing = $state(false)
	// a number input binds a number, a text input a string
	let draft = $state<string | number>("")
	let box: HTMLInputElement | undefined = $state()

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
		const text = String(draft).trim()
		if (isText) {
			onWrite("value", { value: 0, text })
			return
		}
		const n = Number(text)
		if (text !== "" && Number.isFinite(n)) {
			onWrite("value", { value: n, text: "" })
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
		<button class="num settable" type="button" aria-label="Edit {label}" onclick={begin}
			>{text}</button
		>
	{:else}
		<span class="num">{text}</span>
	{/if}
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

	.settable {
		padding: 0 4px;
		margin: 0 -4px;
		border: none;
		border-bottom: 2px dashed var(--border);
		border-radius: 4px 4px 0 0;
		background: none;
		font: inherit;
		font-size: 32px;
		font-weight: 650;
		color: inherit;
		cursor: text;
	}

	.settable:hover {
		background: var(--hover);
	}

	.box {
		width: min(12em, 100%);
		font-size: 24px;
		font-weight: 600;
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
