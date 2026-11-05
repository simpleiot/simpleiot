<script lang="ts">
	// A drawing from a file node, with the elements it marks following
	// their bindings. The markup is sanitized and injected once per
	// drawing; a value change touches attributes in place, so the
	// drawing's own styles and animations carry on. A click on an element
	// marked toggle or set makes the widget a control.
	import Confirm from "../Confirm.svelte"
	import { wantsConfirm, type WidgetProps, type Write } from "../registry.ts"
	import { applyBindings, clickTarget, parseDrawing } from "./drawing.ts"

	let { label, values, options, width, onWrite, writable = [] }: WidgetProps = $props()

	const source = $derived(values.svg)
	const drawing = $derived(source?.state === "live" ? parseDrawing(source.text) : parseDrawing(""))
	const placeholder = $derived(
		!source || source.state === "dead"
			? "bind a file node holding an SVG"
			: source.state === "stale"
				? "the file has no data yet"
				: drawing.error
	)

	// the bindings a click may write through; the host gives no onWrite in
	// edit mode, so a click then does nothing
	const writes = $derived(onWrite ? new Set(writable) : new Set<string>())

	let host: HTMLDivElement | undefined = $state()

	// the drawing is in the DOM once {@html} has placed it, which is before
	// an effect runs; the effect re-runs when the drawing or a value changes
	$effect(() => {
		if (host && drawing.svg !== "") {
			applyBindings(host, values, writes)
		}
	})

	// pending is a write waiting to be confirmed
	let pending = $state<{ prop: string; write: Write } | null>(null)

	function onclick(e: MouseEvent) {
		if (!host || !onWrite) {
			return
		}
		const c = clickTarget(host, e.target as Element | null, values, writes)
		if (!c) {
			return
		}
		e.stopPropagation()
		if (wantsConfirm(options)) {
			pending = c
		} else {
			onWrite(c.prop, c.write)
		}
	}

	const describe = (w: Write) => (w.text !== "" ? w.text : String(w.value))
</script>

<div
	class="drawing"
	class:sized={width !== undefined}
	style:width={width ? `${width}px` : undefined}
>
	{#if drawing.svg !== ""}
		<!-- the click lands on an element inside the drawing, which is what
		     the author made clickable; a keyboard path into a drawing is not
		     offered yet -->
		<!-- eslint-disable svelte/no-at-html-tags -- sanitized by parseDrawing -->
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div class="svg" bind:this={host} {onclick}>
			{@html drawing.svg}
		</div>
		<!-- eslint-enable svelte/no-at-html-tags -->
	{:else}
		<div class="placeholder">
			<span>drawing</span>
			{#if label}<span class="label">{label}</span>{/if}
			<span class="hint">{placeholder}</span>
		</div>
	{/if}
	{#if pending !== null}
		<Confirm
			text="Set {pending.prop} to {describe(pending.write)}?"
			onConfirm={() => {
				const next = pending
				pending = null
				if (next) {
					onWrite?.(next.prop, next.write)
				}
			}}
			onCancel={() => (pending = null)}
		/>
	{/if}
</div>

<style>
	/* in a grid tile the drawing fits the tile both ways; on a canvas it
	   takes the width it is given and its viewBox sets the height */
	.drawing {
		height: 100%;
	}

	.svg {
		height: 100%;
	}

	.svg :global(svg) {
		display: block;
		width: 100%;
		height: 100%;
	}

	.sized,
	.sized .svg {
		height: auto;
	}

	.sized .svg :global(svg) {
		height: auto;
	}

	.placeholder {
		display: flex;
		flex-direction: column;
		gap: 4px;
		color: var(--muted);
	}

	.label {
		font-weight: 600;
	}

	.hint {
		font-size: 13px;
	}
</style>
