<script lang="ts">
	// FahrenheitInput is a checkbox that sets the units point to "F" when
	// checked and "C" when not.
	import { editTime, inputContext } from "#lib/inputs/context.ts"
	import Label from "#lib/inputs/Label.svelte"
	import { getText, newText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"

	let { label }: { label: string } = $props()

	const ctx = inputContext()
	const id = $props.id()

	function onchange(e: Event & { currentTarget: HTMLInputElement }) {
		const t = e.currentTarget.checked ? "F" : "C"
		ctx.onEditPoints([newText(PointType.Units, "0", t, editTime(ctx))])
	}
</script>

<Label {label} {id}>
	<input
		{id}
		type="checkbox"
		checked={getText(ctx.node.points, PointType.Units) === "F"}
		{onchange}
	/>
</Label>
