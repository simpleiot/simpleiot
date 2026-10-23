<script lang="ts">
	import { blankMagicValue, DataType, get } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"

	let {
		type,
		key = "0",
		label = "",
		placeholder = "",
	}: { type: string; key?: string; label?: string; placeholder?: string } = $props()

	const ctx = inputContext()
	const id = $props.id()

	// a number point shows empty here; the number input edits it
	const value = $derived.by(() => {
		const p = get(ctx.node.points, type, key)
		if (!p || p.text === blankMagicValue || p.value !== 0) {
			return ""
		}
		return p.text
	})

	function oninput(e: Event & { currentTarget: HTMLInputElement }) {
		ctx.onEditPoints([
			{
				type,
				key,
				time: editTime(ctx),
				dataType: DataType.String,
				value: 0,
				text: e.currentTarget.value,
				tombstone: 0,
			},
		])
	}
</script>

<Label {label} {id} edited={ctx.edited(type, key)}>
	<input
		{id}
		type="text"
		{value}
		{placeholder}
		{oninput}
		aria-label={label === "" ? placeholder || type : undefined}
	/>
</Label>
