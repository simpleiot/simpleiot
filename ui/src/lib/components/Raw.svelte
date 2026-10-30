<script lang="ts">
	// Raw shows a node as its points, for node types with no editor and
	// for the raw view of any node.
	import X from "@lucide/svelte/icons/x"
	import { editTime, setInputContext } from "#lib/inputs/context.ts"
	import RawInput from "#lib/inputs/RawInput.svelte"
	import type { EditorProps } from "#lib/nodes/types.ts"
	import { filterDeleted, getText, newTombstone, sorted } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import IconButton from "./IconButton.svelte"

	let props: EditorProps = $props()

	const ctx = {
		get node() {
			return props.node
		},
		get now() {
			return props.now
		},
		labelWidth: 0,
		get scratch() {
			return props.scratch
		},
		edited: (type: string, key?: string) =>
			props.pending.some((p) => p.type === type && (key === undefined || p.key === key)),
		onEditPoints: (points: EditorProps["node"]["points"]) => {
			props.onEditPoints(points)
		},
		onEditScratch: (text: string) => {
			props.onEditScratch(text)
		},
	}
	setInputContext(ctx)

	const points = $derived(sorted(filterDeleted(props.node.points)))
</script>

<div class="editor">
	<div class="header">
		<span class="type">Node type: {props.node.type}</span>
		<span>{getText(props.node.points, PointType.Description)}</span>
	</div>
	{#if props.expDetail}
		<div class="id">ID: <span class="mono">{props.node.id}</span></div>
		<div class="table-wrap">
			<table>
				<thead>
					<tr><th>Point</th><th>Value</th><th><span class="visually-hidden">Delete</span></th></tr>
				</thead>
				<tbody>
					{#each points as p (`${p.type}.${p.key}`)}
						<tr>
							<td>{p.type}:{p.key}</td>
							<td><RawInput type={p.type} key={p.key} /></td>
							<td>
								<IconButton
									label="Delete point {p.type}:{p.key}"
									onclick={() => props.onEditPoints([newTombstone(p.type, p.key, editTime(ctx))])}
								>
									<X size={20} />
								</IconButton>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style>
	.editor {
		border-top: 2px solid var(--fg);
		padding-top: 8px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px 10px;
		min-height: 34px;
	}

	.type {
		background: var(--highlight-bg);
		padding: 0 4px;
	}

	.id {
		margin: 0 7px;
	}

	.mono {
		font-family: monospace;
		user-select: all;
	}

	.table-wrap {
		overflow-x: auto;
	}

	table {
		border-collapse: collapse;
		margin: 7px;
	}

	th,
	td {
		padding: 5px 15px;
		text-align: left;
	}

	td :global(input) {
		width: 24em;
	}
</style>
