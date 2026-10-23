<script lang="ts">
	// NumberInput edits a number point. While editing, the text typed is
	// kept in the point's text so "1." or "-" survive until saved; an
	// empty field holds blankMagicValue.
	import { blankMagicValue, DataType, get } from "#lib/siot/point.ts"
	import { float } from "#lib/util/sanitize.ts"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"

	let { type, key = "0", label = "" }: { type: string; key?: string; label?: string } = $props()

	const ctx = inputContext()
	const id = $props.id()

	const point = $derived(get(ctx.node.points, type, key))

	const value = $derived.by(() => {
		const p = point
		if (!p) {
			return ""
		}
		if (p.text !== "") {
			if (p.text === blankMagicValue || p.text === "blank") {
				return ""
			}
			if (p.text === "-") {
				return "-"
			}
			return float(p.text)
		}
		return String(Math.round(p.value * 1e6) / 1e6)
	})

	function oninput(e: Event & { currentTarget: HTMLInputElement }) {
		const d = e.currentTarget.value
		let text: string
		if (d === "") {
			text = blankMagicValue
		} else if (d === "-") {
			text = "-"
		} else if (Number.isFinite(Number(d)) && d.trim() !== "") {
			text = d
		} else {
			// not a number: keep what was there
			text = value
			e.currentTarget.value = value
		}
		const v = text === blankMagicValue || text === "-" ? 0 : Number(text)
		ctx.onEditPoints([
			{
				type,
				key,
				time: editTime(ctx),
				dataType: DataType.Float,
				value: Number.isFinite(v) ? v : (point?.value ?? 0),
				text,
				tombstone: 0,
			},
		])
	}
</script>

<Label {label} {id}>
	<input
		{id}
		type="text"
		inputmode="decimal"
		{value}
		{oninput}
		aria-label={label === "" ? type : undefined}
	/>
</Label>
