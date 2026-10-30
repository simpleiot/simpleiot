<script lang="ts">
	// CounterWithReset shows a count, and a box to ask the client to reset
	// it.
	import { DataType, getValue } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"

	let {
		type,
		resetType,
		key = "0",
		label,
	}: { type: string; resetType: string; key?: string; label: string } = $props()

	const ctx = inputContext()
	const id = $props.id()

	function onchange(e: Event & { currentTarget: HTMLInputElement }) {
		ctx.onEditPoints([
			{
				type: resetType,
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

<div class="counter">
	<span class="count" style:width="{ctx.labelWidth}px">
		{label}: {getValue(ctx.node.points, type, key)}
	</span>
	<label for={id}>reset</label>
	<input
		{id}
		type="checkbox"
		checked={getValue(ctx.node.points, resetType, key) !== 0}
		{onchange}
	/>
</div>

<style>
	.counter {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px 10px;
	}

	.count {
		text-align: right;
		margin-right: 10px;
	}

	@media (max-width: 600px) {
		.count {
			text-align: left;
			width: auto !important;
		}
	}
</style>
