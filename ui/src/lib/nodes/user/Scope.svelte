<script lang="ts">
	// Scope gives the inputs inside it a changed view of the node, or
	// changes their edits before they are recorded, as the Elm editor did
	// by handing an input its own options.
	import type { Snippet } from "svelte"
	import { inputContext, setInputContext } from "#lib/inputs/context.ts"
	import type { Point } from "#lib/siot/point.ts"

	type MapPoints = (points: Point[]) => Point[]

	let {
		view = (points) => points,
		edit = (points) => points,
		children,
	}: {
		// view changes the points the inputs show
		view?: MapPoints
		// edit changes the points the inputs record
		edit?: MapPoints
		children: Snippet
	} = $props()

	const ctx = inputContext()
	const node = $derived({ ...ctx.node, points: view(ctx.node.points) })

	setInputContext({
		get node() {
			return node
		},
		get now() {
			return ctx.now
		},
		get labelWidth() {
			return ctx.labelWidth
		},
		get scratch() {
			return ctx.scratch
		},
		edited: (type, key) => ctx.edited(type, key),
		onEditPoints: (points) => {
			ctx.onEditPoints(edit(points))
		},
		onEditScratch: (text) => {
			ctx.onEditScratch(text)
		},
	})
</script>

{@render children()}
