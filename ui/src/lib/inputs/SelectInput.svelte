<script lang="ts">
	// SelectInput picks one text value from a drop-down: the fixed options,
	// then any extra values under their own heading, such as the point
	// types a chosen node carries. A stored value outside both is kept as
	// an option so it is never lost.
	import { DataType, getText } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"

	let {
		type,
		key = "0",
		label,
		options,
		extra = [],
		extraLabel = "on this node",
	}: {
		type: string
		key?: string
		label: string
		// options are [value, label] pairs
		options: readonly (readonly [string, string])[]
		extra?: readonly string[]
		extraLabel?: string
	} = $props()

	const ctx = inputContext()
	const id = $props.id()
	const selected = $derived(getText(ctx.node.points, type, key))
	const more = $derived(extra.filter((v) => !options.some(([o]) => o === v)))
	const unknown = $derived(
		selected !== "" && !options.some(([o]) => o === selected) && !more.includes(selected)
	)

	function onchange(e: Event & { currentTarget: HTMLSelectElement }) {
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
	<select {id} value={selected} {onchange}>
		<option value="">(none)</option>
		{#each options as [value, text] (value)}
			<option {value}>{text}</option>
		{/each}
		{#if more.length > 0}
			<optgroup label={extraLabel}>
				{#each more as value (value)}
					<option {value}>{value}</option>
				{/each}
			</optgroup>
		{/if}
		{#if unknown}
			<option value={selected}>{selected}</option>
		{/if}
	</select>
</Label>

<style>
	select {
		padding: 6px 8px;
		font: inherit;
		color: var(--fg);
		background: var(--input-bg);
		border: 1px solid var(--border);
		border-radius: 4px;
	}
</style>
