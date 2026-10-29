<script lang="ts">
	// RawInput edits any point as the raw view shows it: a string in
	// quotes, a number bare. Text that is neither is kept in the field and
	// marked invalid until it is fixed.
	import { formatRaw, get, parseRaw } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"

	let { type, key = "0", label = "" }: { type: string; key?: string; label?: string } = $props()

	const ctx = inputContext()
	const id = $props.id()

	const point = $derived(get(ctx.node.points, type, key))

	// draft is the text typed since the field last showed the point
	let draft = $state<string | null>(null)
	let invalid = $state(false)

	const value = $derived(draft ?? (point ? formatRaw(point) : ""))

	function oninput(e: Event & { currentTarget: HTMLInputElement }) {
		draft = e.currentTarget.value
		const p = parseRaw(draft, point, type, key, editTime(ctx))
		invalid = p === undefined
		if (p) {
			ctx.onEditPoints([p])
		}
	}

	function onblur() {
		if (!invalid) {
			draft = null
		}
	}
</script>

<Label {label} {id} edited={ctx.edited(type, key)}>
	<input
		{id}
		type="text"
		class:invalid
		{value}
		{oninput}
		{onblur}
		aria-invalid={invalid}
		aria-label={label === "" ? type : undefined}
	/>
</Label>

<style>
	.invalid {
		outline: 2px solid var(--red);
	}
</style>
