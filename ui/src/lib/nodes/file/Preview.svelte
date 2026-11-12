<script lang="ts">
	// Preview shows a file that holds an image. An SVG is drawn as the
	// dashboard draws it, and when it declares bindings each gets a test
	// value, so the drawing can be tried without a dashboard: type a value,
	// or click an element marked toggle or set. Nothing is sent.
	import Label from "#lib/inputs/Label.svelte"
	import { imageSource, svgText } from "#lib/widgets/image/image.ts"
	import type { Bound, Write } from "#lib/widgets/registry.ts"
	import { parseDrawing } from "#lib/widgets/svg/drawing.ts"
	import DrawingWidget from "#lib/widgets/svg/Widget.svelte"
	import { testBound, writeText } from "./preview.ts"

	let { data, label }: { data: string; label: string } = $props()

	const svg = $derived(svgText(data))
	const drawing = $derived(svg === "" ? undefined : parseDrawing(svg))
	const image = $derived(svg === "" ? imageSource(data) : undefined)

	const tests = $state<Record<string, string>>({})
	const names = $derived(drawing?.bindings.map((b) => b.name) ?? [])
	const values = $derived.by(() => {
		const out: Record<string, Bound> = {
			svg: { state: "live", value: 0, text: svg, time: "" },
		}
		for (const name of names) {
			out[name] = testBound(tests[name] ?? "")
		}
		return out
	})

	// the preview's width, less its padding, is what the drawing takes
	let width = $state(0)
	const id = $props.id()
</script>

{#if drawing}
	<div class="preview" bind:clientWidth={width}>
		{#if width > 16}
			<DrawingWidget
				{label}
				{values}
				options={{}}
				width={width - 16}
				writable={names}
				onWrite={(prop: string, w: Write) => (tests[prop] = writeText(w))}
			/>
		{/if}
	</div>
	{#if drawing.bindings.length > 0}
		<div class="hint">
			Test values exercise the drawing here and are not sent; clicking an element marked toggle or
			set changes them too. An empty value is stale.
		</div>
		{#each drawing.bindings as b, i (b.name)}
			<Label label={b.name} id="{id}-{i}">
				<input
					id="{id}-{i}"
					type="text"
					value={tests[b.name] ?? ""}
					oninput={(e) => (tests[b.name] = e.currentTarget.value)}
				/>
				<!-- what the drawing does with the binding, as the widget editor
				     lists it -->
				<span class="use">{b.use}</span>
			</Label>
		{/each}
	{/if}
{:else if image && image.src !== ""}
	<div class="preview">
		<img src={image.src} alt={label} />
	</div>
{/if}

<style>
	.preview {
		max-width: 480px;
		padding: 8px;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--bg);
	}

	img {
		display: block;
		max-width: 100%;
		max-height: 480px;
	}

	.use {
		color: var(--muted);
		font-size: 13px;
	}

	.hint {
		font-size: 13px;
		color: var(--muted);
	}
</style>
