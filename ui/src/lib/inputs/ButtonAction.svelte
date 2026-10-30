<script lang="ts">
	// ButtonAction edits a point to a fixed number or text when pressed, for
	// actions a client carries out and then clears, such as resetting a
	// value.
	import { DataType } from "#lib/siot/point.ts"
	import Button from "./Button.svelte"
	import type { Color } from "./color.ts"
	import { editTime, inputContext } from "./context.ts"

	let {
		type,
		key = "0",
		value = 0,
		text,
		label,
		color = "blue",
	}: {
		type: string
		key?: string
		value?: number
		// text sends a text point instead of value
		text?: string
		label: string
		color?: Color
	} = $props()

	const ctx = inputContext()

	function onclick() {
		ctx.onEditPoints([
			{
				type,
				key,
				time: editTime(ctx),
				dataType: text === undefined ? DataType.Float : DataType.String,
				value: text === undefined ? value : 0,
				text: text ?? "",
				tombstone: 0,
			},
		])
	}
</script>

<Button {color} {onclick}>{label}</Button>
