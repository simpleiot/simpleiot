<script lang="ts">
	// ListInput edits a list of text kept as one point per index.
	import X from "@lucide/svelte/icons/x"
	import IconButton from "#lib/components/IconButton.svelte"
	import { DataType, getTextArray, newTombstone, type Point } from "#lib/siot/point.ts"
	import Button from "./Button.svelte"
	import Label from "./Label.svelte"
	import { editTime, inputContext } from "./context.ts"

	let { type, label, buttonLabel }: { type: string; label: string; buttonLabel: string } = $props()

	const ctx = inputContext()

	const entries = $derived(getTextArray(ctx.node.points, type))
	// count includes deleted entries, so a shorter list deletes them all
	const count = $derived(ctx.node.points.filter((p) => p.type === type).length)

	function toPoints(list: string[]): Point[] {
		const time = editTime(ctx)
		const pts: Point[] = list.map((text, i) => ({
			type,
			key: String(i),
			time,
			dataType: DataType.String,
			value: 0,
			text,
			tombstone: 0,
		}))
		for (let i = list.length; i < count; i++) {
			pts.push(newTombstone(type, String(i), time))
		}
		return pts
	}

	function update(i: number, text: string) {
		ctx.onEditPoints(toPoints(entries.map((e, j) => (j === i ? text : e))))
	}

	function remove(i: number) {
		ctx.onEditPoints(toPoints(entries.filter((_, j) => j !== i)))
	}

	function add() {
		ctx.onEditPoints([
			{
				type,
				key: String(count),
				time: editTime(ctx),
				dataType: DataType.String,
				value: 0,
				text: "",
				tombstone: 0,
			},
		])
	}
</script>

<Label {label} top edited={ctx.edited(type)}>
	<div class="list">
		{#each entries as entry, i (i)}
			<div class="entry">
				<input
					type="text"
					aria-label="{label} {i + 1}"
					value={entry}
					oninput={(e) => {
						update(i, e.currentTarget.value)
					}}
				/>
				<IconButton
					label="Remove {label} {i + 1}"
					onclick={() => {
						remove(i)
					}}><X size={20} /></IconButton
				>
			</div>
		{/each}
		<div><Button onclick={add}>{buttonLabel}</Button></div>
	</div>
</Label>

<style>
	.list {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.entry {
		display: flex;
		align-items: center;
		gap: 10px;
	}
</style>
