<script lang="ts">
	// KeyValueInput edits text points of one type keyed by name, such as
	// tags. The key for a new entry is typed into the editor's scratch.
	import X from "@lucide/svelte/icons/x"
	import IconButton from "#lib/components/IconButton.svelte"
	import { DataType, filterDeleted, getAll, newTombstone, sorted } from "#lib/siot/point.ts"
	import Button from "./Button.svelte"
	import { editTime, inputContext } from "./context.ts"
	import TextInput from "./TextInput.svelte"

	let { type, label, buttonLabel }: { type: string; label: string; buttonLabel: string } = $props()

	const ctx = inputContext()
	const points = $derived(sorted(filterDeleted(getAll(ctx.node.points, type))))

	function add() {
		ctx.onEditPoints([
			{
				type,
				key: ctx.scratch,
				time: editTime(ctx),
				dataType: DataType.String,
				value: 0,
				text: "",
				tombstone: 0,
			},
		])
	}
</script>

<div class="kv">
	{#if points.length > 0}
		<strong>{label}</strong>
		<table>
			<thead>
				<tr><th>Key</th><th>Value</th><th>Delete</th></tr>
			</thead>
			<tbody>
				{#each points as p (p.key)}
					<tr>
						<td>{p.key}:</td>
						<td><TextInput {type} key={p.key} placeholder="value" /></td>
						<td class="center">
							<IconButton
								label="Delete {p.key}"
								onclick={() => {
									ctx.onEditPoints([newTombstone(type, p.key, editTime(ctx))])
								}}><X size={20} /></IconButton
							>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
	<div class="add">
		<span>Add:</span>
		<input
			type="text"
			placeholder="new key"
			aria-label="New {label} key"
			value={ctx.scratch}
			oninput={(e) => {
				ctx.onEditScratch(e.currentTarget.value)
			}}
			onkeyup={(e) => {
				if (e.key === "Enter") {
					e.stopPropagation()
					add()
				}
			}}
		/>
		<Button onclick={add}>{buttonLabel}</Button>
	</div>
</div>

<style>
	.kv {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 5px;
	}

	strong {
		padding: 6px 0;
	}

	table {
		border-collapse: collapse;
		margin: 7px;
	}

	th,
	td {
		padding: 5px 15px;
	}

	.center {
		text-align: center;
	}

	.add {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}

	.add span {
		padding-left: 40px;
	}

	@media (max-width: 600px) {
		.add span {
			padding-left: 0;
		}
	}
</style>
