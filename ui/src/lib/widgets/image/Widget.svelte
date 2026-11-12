<script lang="ts">
	// A picture from a file node: a site photo, a panel, a logo. On a
	// canvas it takes the width it is given at its own proportions, so it
	// can sit behind the equipment drawn over it; in a grid tile the fit
	// option says how it fills the tile.
	import { num, type WidgetProps } from "../registry.ts"
	import { imageSource } from "./image.ts"

	let { label, values, options, width }: WidgetProps = $props()

	const source = $derived(values.image)
	const image = $derived(source?.state === "live" ? imageSource(source.text) : imageSource(""))
	const placeholder = $derived(
		!source || source.state === "dead"
			? "bind a file node holding an image"
			: source.state === "stale"
				? "the file has no data yet"
				: image.error
	)
	const fits: Readonly<Record<string, string>> = {
		contain: "contain",
		cover: "cover",
		stretch: "fill",
	}
	const objectFit = $derived(fits[options.fit ?? ""] ?? "contain")
	const opacity = $derived(Math.min(1, Math.max(0, num(options, "opacity", 1))))
</script>

<div class="image" class:sized={width !== undefined} style:width={width ? `${width}px` : undefined}>
	{#if image.src !== ""}
		<img src={image.src} alt={label} style:object-fit={objectFit} style:opacity draggable="false" />
	{:else}
		<div class="placeholder">
			<span>image</span>
			{#if label}<span class="label">{label}</span>{/if}
			<span class="hint">{placeholder}</span>
		</div>
	{/if}
</div>

<style>
	/* in a grid tile the image fills the tile as the fit option says; on
	   a canvas it takes the width and its own proportions set the height */
	.image {
		height: 100%;
	}

	img {
		display: block;
		width: 100%;
		height: 100%;
	}

	.sized {
		height: auto;
	}

	.sized img {
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
