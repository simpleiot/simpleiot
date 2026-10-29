<script lang="ts">
	// NodeRow is one node in the tree: the expand and detail toggles, the
	// node's editor, its pending edit, and when its detail is open the
	// operation panel; then its children.
	import ChevronDown from "@lucide/svelte/icons/chevron-down"
	import ChevronRight from "@lucide/svelte/icons/chevron-right"
	import Loader from "@lucide/svelte/icons/loader"
	import Blank from "#lib/icons/Blank.svelte"
	import { registry } from "#lib/nodes/index.ts"
	import type { EditorProps } from "#lib/nodes/types.ts"
	import { edgeRole, isTombstone, nodeStatus } from "#lib/siot/node.ts"
	import { getBestDesc, updatePoints, type Point } from "#lib/siot/point.ts"
	import { store } from "#lib/siot/store.ts"
	import type { TreeNode } from "#lib/siot/tree.ts"
	import IconButton from "./IconButton.svelte"
	import NodeOps from "./NodeOps.svelte"
	import NodeRow from "./NodeRow.svelte"
	import Raw from "./Raw.svelte"

	let { t, parent }: { t: TreeNode; parent: TreeNode | undefined } = $props()

	const editing = $derived(store.edit?.key === t.key ? store.edit : null)
	const node = $derived(
		editing && editing.points.length > 0
			? { ...t.node, points: updatePoints(t.node.points, editing.points) }
			: t.node
	)
	const viewRaw = $derived(editing?.viewRaw ?? false)
	const Editor = $derived(viewRaw ? Raw : (registry.editorFor(t.node.type) ?? Raw))
	// while searching, the children shown are the ones the search keeps
	const filter = $derived(store.searchKeys)
	const childRows = $derived(
		filter
			? t.children.filter((c) => filter.has(c.key) && !isTombstone(c.node))
			: t.expChildren
				? t.children.filter((c) => !isTombstone(c.node))
				: []
	)
	const open = $derived(filter ? childRows.length > 0 : t.expChildren)
	const desc = $derived(registry.get(t.node.type)?.label ?? t.node.type)
	const status = $derived(nodeStatus(node))
	const statusText = { ok: "", disabled: " (disabled)", error: " (error)", offline: " (offline)" }

	let fileInput: HTMLInputElement | undefined = $state()
	let uploadBinary = false

	const editorProps: EditorProps = $derived({
		node,
		typeLabel: desc,
		parent: parent?.node,
		children: t.children,
		now: store.now,
		zone: store.zone,
		expDetail: t.expDetail,
		modified: editing !== null,
		pending: editing?.points ?? [],
		copy: store.clipboard,
		scratch: store.scratch,
		generatedToken: store.generatedToken,
		findNode: (id: string) => store.findNode(id),
		onEditPoints: (points: Point[]) => {
			store.editPoints(t.key, points)
		},
		onSendPoints: (points: Point[]) => void store.sendPoints(t.node.id, points),
		onEditScratch: (text: string) => (store.scratch = text),
		onUploadFile: (binary: boolean) => {
			uploadBinary = binary
			fileInput?.click()
		},
		onGenerateKey: () => void store.generateKey(t.node.id),
	})

	// Enter in the node's form saves its edit; Escape cancels an open
	// operation, or else discards the edit.
	function onkeyup(e: KeyboardEvent) {
		const target = e.target as HTMLElement
		if (target.tagName === "TEXTAREA" || target.tagName === "BUTTON") {
			return
		}
		if (e.key === "Enter" && editing) {
			store.save(t.key)
		} else if (e.key === "Escape") {
			if (store.op.kind !== "none" && store.op.key === t.key) {
				store.cancelOp()
			} else if (editing) {
				store.discardEdits()
			}
		}
	}

	function onFile(e: Event & { currentTarget: HTMLInputElement }) {
		const file = e.currentTarget.files?.[0]
		e.currentTarget.value = ""
		if (file) {
			void store.uploadFile(t.node.id, uploadBinary, file)
		}
	}
</script>

<li
	role="treeitem"
	aria-expanded={t.hasChildren ? open : undefined}
	aria-selected={t.expDetail}
	tabindex={store.tabKey === t.key ? 0 : -1}
	aria-label="{desc}: {getBestDesc(node.points)}"
	data-key={t.key}
	class="item"
>
	<div class="row">
		<div class="toggles">
			{#if t.loading}
				<span class="spin" aria-label="loading"><Loader size={22} /></span>
			{:else if !t.hasChildren}
				<span class="pad"><Blank size={22} /></span>
			{:else}
				<IconButton
					label={open ? "Collapse" : "Expand"}
					disabled={filter !== null}
					onclick={() => void store.toggleChildren(t.key)}
				>
					{#if open}<ChevronDown size={22} />{:else}<ChevronRight size={22} />{/if}
				</IconButton>
			{/if}
			<IconButton
				label={(t.expDetail ? "Close details" : "Open details") + statusText[status]}
				pressed={t.expDetail}
				onclick={() => store.toggleDetail(t.key)}
			>
				<!-- the dot's color is the node's status -->
				<svg class="dot {status}" width="22" height="22" viewBox="0 0 24 24" aria-hidden="true">
					<circle cx="12" cy="12" r="4.1" fill="currentColor" />
				</svg>
			</IconButton>
		</div>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="body"
			class:detail={t.expDetail}
			class:modified={store.hasEdits && editing !== null}
			{onkeyup}
		>
			{#if edgeRole(t.node) === "mirror"}
				<span class="badge mirror">mirror</span>
			{/if}
			<Editor {...editorProps} />
			{#if viewRaw}
				<div class="new-point">
					<label>
						New point type:
						<input type="text" bind:value={store.addPoint.type} />
					</label>
					<label>
						New point key:
						<input type="text" bind:value={store.addPoint.key} />
					</label>
				</div>
			{/if}
			{#if editing}
				<!-- stays in view while the editor scrolls, until saved or discarded -->
				<div class="save-bar">
					{#if editing.points.length > 0}
						<span class="unsaved">Unsaved changes</span>
					{/if}
					<button class="btn blue" onclick={() => store.save(t.key)}>save</button>
					<button class="btn gray" onclick={() => store.discardEdits()}>discard</button>
					{#if viewRaw}
						<button class="btn green" onclick={() => store.addRawPoint(t.key)}>add point</button>
					{/if}
				</div>
			{/if}
			{#if t.expDetail}
				<NodeOps {t} />
			{/if}
			<input
				type="file"
				class="visually-hidden"
				tabindex="-1"
				bind:this={fileInput}
				onchange={onFile}
			/>
		</div>
	</div>
	{#if childRows.length > 0}
		<ul role="group">
			{#each childRows as c (c.key)}
				<NodeRow t={c} parent={t} />
			{/each}
		</ul>
	{/if}
</li>

<style>
	.item {
		list-style: none;
	}

	/* the focus ring goes round the row, not the subtree under it */
	.item:focus-visible {
		outline: none;
	}

	.item:focus-visible > .row {
		outline: 3px solid var(--focus);
		outline-offset: 1px;
		border-radius: 2px;
	}

	.row {
		display: flex;
		gap: 6px;
	}

	.toggles {
		display: flex;
		align-items: flex-start;
		gap: 6px;
		padding-top: 18px;
		flex-shrink: 0;
	}

	.dot.error {
		color: var(--red);
	}

	.dot.offline {
		color: var(--orange);
	}

	.dot.disabled {
		color: var(--muted);
	}

	.pad,
	.spin {
		display: inline-flex;
		padding: 5px;
	}

	.spin :global(svg) {
		animation: spin 1.2s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 6px;
	}

	.body.detail {
		background: var(--detail-bg);
	}

	.body.modified {
		box-shadow: inset 3px 0 0 var(--blue);
	}

	.save-bar {
		position: sticky;
		bottom: 0;
		z-index: 1;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 16px;
		padding: 10px 16px;
		background: var(--detail-bg);
		border-top: 1px solid var(--border);
	}

	.unsaved {
		color: var(--blue);
		font-weight: 600;
	}

	.mirror {
		align-self: flex-start;
	}

	.new-point {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.new-point label {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}

	ul {
		margin: 0;
		padding: 0 0 0 35px;
	}

	@media (max-width: 600px) {
		ul {
			padding-left: 14px;
		}

		.row,
		.toggles {
			gap: 0;
		}

		.body {
			padding: 6px 4px;
		}

		.save-bar {
			gap: 6px 16px;
			padding: 8px;
		}

		.unsaved {
			flex-basis: 100%;
		}
	}
</style>
