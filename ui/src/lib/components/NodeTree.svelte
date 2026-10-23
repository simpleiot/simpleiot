<script lang="ts">
	// NodeTree shows the forest, one tree per group the user belongs to. It
	// is an ARIA tree with one tab stop: the arrow keys move between rows,
	// Right and Left expand and collapse, Enter opens a node's detail, and
	// Escape cancels an open operation.
	import { tick } from "svelte"
	import { store } from "#lib/siot/store.ts"
	import { parentKey } from "#lib/siot/tree.ts"
	import NodeRow from "./NodeRow.svelte"

	async function focusRow(key: string | undefined) {
		if (!key) {
			return
		}
		store.focusKey = key
		await tick()
		const el = document.querySelector<HTMLElement>(
			`[role="treeitem"][data-key="${CSS.escape(key)}"]`
		)
		el?.focus()
		el?.scrollIntoView({ block: "nearest" })
	}

	function onkeydown(e: KeyboardEvent) {
		const el = e.target as HTMLElement
		// keys typed into a row's inputs are the editor's
		if (el.getAttribute("role") !== "treeitem") {
			return
		}
		const key = el.dataset.key ?? ""
		const rows = store.rows
		const i = rows.findIndex((r) => r.key === key)
		const row = rows[i]
		if (!row) {
			return
		}
		const showing = rows[i + 1] !== undefined && parentKey(rows[i + 1]?.key ?? "") === key
		switch (e.key) {
			case "ArrowDown":
				void focusRow(rows[i + 1]?.key)
				break
			case "ArrowUp":
				void focusRow(rows[i - 1]?.key)
				break
			case "Home":
				void focusRow(rows[0]?.key)
				break
			case "End":
				void focusRow(rows[rows.length - 1]?.key)
				break
			case "ArrowRight":
				if (showing) {
					void focusRow(rows[i + 1]?.key)
				} else if (row.hasChildren && !store.searchKeys) {
					void store.expand(key)
				}
				break
			case "ArrowLeft":
				if (showing && !store.searchKeys) {
					void store.toggleChildren(key)
				} else {
					void focusRow(parentKey(key) || undefined)
				}
				break
			case "Enter":
				store.toggleDetail(key)
				break
			case "Escape":
				if (store.op.kind !== "none" && store.op.key === key) {
					store.cancelOp()
				}
				break
			default:
				return
		}
		e.preventDefault()
	}

	function onfocusin(e: FocusEvent) {
		const li = (e.target as HTMLElement).closest<HTMLElement>('[role="treeitem"]')
		if (li?.dataset.key) {
			store.focusKey = li.dataset.key
		}
	}
</script>

<ul class="tree" role="tree" aria-label="Nodes" {onkeydown} {onfocusin}>
	{#each store.forest as t (t.key)}
		{#if !store.searchKeys || store.searchKeys.has(t.key)}
			<NodeRow {t} parent={undefined} />
		{/if}
	{/each}
</ul>
{#if store.searchKeys && store.searchKeys.size === 0 && !store.searching}
	<p class="none" role="status">No nodes match “{store.query}”.</p>
{/if}

<style>
	.tree {
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.none {
		margin: 0;
		color: var(--muted);
	}
</style>
