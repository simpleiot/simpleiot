<script lang="ts">
	// OptionInput picks one of a set of text values.
	import { DataType, getText } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"

	let {
		type,
		key = "0",
		label,
		options,
	}: {
		type: string
		key?: string
		label: string
		// options are [value, label] pairs
		options: readonly (readonly [string, string])[]
	} = $props()

	const ctx = inputContext()
	const name = $props.id()
	const selected = $derived(getText(ctx.node.points, type, key))

	function choose(value: string) {
		ctx.onEditPoints([
			{
				type,
				key,
				time: editTime(ctx),
				dataType: DataType.String,
				value: 0,
				text: value,
				tombstone: 0,
			},
		])
	}
</script>

<fieldset>
	<legend style:width="{ctx.labelWidth}px" class:edited={ctx.edited(type, key)}>{label}:</legend>
	<div class="options">
		{#each options as [value, text] (value)}
			<label>
				<input
					type="radio"
					{name}
					{value}
					checked={selected === value}
					onchange={() => {
						choose(value)
					}}
				/>
				{text}
			</label>
		{/each}
	</div>
</fieldset>

<style>
	.edited {
		color: var(--blue);
		font-weight: 600;
	}

	fieldset {
		border: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 4px 10px;
	}

	legend {
		float: left;
		text-align: right;
		padding: 12px 0;
		flex-shrink: 0;
	}

	.options {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 12px 0;
	}

	label {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	@media (max-width: 600px) {
		legend {
			text-align: left;
			width: auto !important;
			padding-bottom: 0;
		}
	}
</style>
