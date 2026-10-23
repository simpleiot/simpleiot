<script lang="ts">
	import { DataType, getValue } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"

	let { type, key = "0", label = "" }: { type: string; key?: string; label?: string } = $props()

	const ctx = inputContext()
	const id = $props.id()

	function onchange(e: Event & { currentTarget: HTMLInputElement }) {
		ctx.onEditPoints([
			{
				type,
				key,
				time: editTime(ctx),
				dataType: DataType.Float,
				value: e.currentTarget.checked ? 1 : 0,
				text: "",
				tombstone: 0,
			},
		])
	}
</script>

<Label {label} {id} edited={ctx.edited(type, key)}>
	<input
		{id}
		type="checkbox"
		checked={getValue(ctx.node.points, type, key) === 1}
		{onchange}
		aria-label={label === "" ? type : undefined}
	/>
</Label>
