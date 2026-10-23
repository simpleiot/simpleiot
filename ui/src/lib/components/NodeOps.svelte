<script lang="ts">
	// NodeOps is the operation panel under an open node: add a child,
	// message, delete, copy and paste, and the raw view; and the form of
	// whichever operation is in progress.
	import ClipboardIcon from "@lucide/svelte/icons/clipboard"
	import CirclePlus from "@lucide/svelte/icons/circle-plus"
	import Copy from "@lucide/svelte/icons/copy"
	import Link from "@lucide/svelte/icons/link"
	import List from "@lucide/svelte/icons/list"
	import MessageSquare from "@lucide/svelte/icons/message-square"
	import X from "@lucide/svelte/icons/x"
	import { resolve } from "$app/paths"
	import { registry } from "#lib/nodes/index.ts"
	import { edgeRole } from "#lib/siot/node.ts"
	import { store } from "#lib/siot/store.ts"
	import { writeClipboard } from "#lib/siot/tree.svelte.ts"
	import type { TreeNode } from "#lib/siot/tree.ts"
	import IconButton from "./IconButton.svelte"

	let { t }: { t: TreeNode } = $props()

	const op = $derived(store.op.kind !== "none" && store.op.key === t.key ? store.op : null)
	const msg = $derived(store.nodeMsg?.key === t.key ? store.nodeMsg.text : null)
	const canAdd = $derived(registry.hasChildren(t.node.type))
	const addable = $derived(registry.addableUnder(t.node.type))
	const viewRaw = $derived(store.edit?.key === t.key && store.edit.viewRaw)

	const deleteQuestion = $derived(
		{
			// deleting the node where it lives takes its mirrors with it. No
			// count is offered, because mirrors can sit in groups this user
			// cannot see and a count from the visible tree would be wrong.
			primary: "Delete this node, and any mirrors of it?",
			mirror: "Remove this mirror? The node itself is not deleted.",
			none: "Delete this node?",
		}[edgeRole(t.node)]
	)

	function copyLink() {
		writeClipboard(new URL(resolve("/nodes/[id]", { id: t.node.id }), location.href).href)
		store.nodeMsg = { key: t.key, text: "Link copied", time: Date.now() }
	}

	// paste choices: some node types are found by walking down from their
	// parent rather than from the tree root, so a modbusIo dropped into a
	// group stops working quietly. Mirroring is what was wanted in that
	// case, and it is all that is offered.
	const paste = $derived.by(() => {
		const c = store.clipboard
		if (!c) {
			return { text: "Select node to copy/move first", ops: [] as const }
		}
		if (c.id === t.node.id) {
			return { text: "Can't move/copy node to itself", ops: [] as const }
		}
		if (c.parent === t.node.id) {
			return { text: `Copy ${c.description} here?`, ops: ["duplicate"] as const }
		}
		const owner = store.clipboardOwner()
		if (owner === undefined || owner === t.node.type) {
			return { text: `Copy ${c.description} here?`, ops: ["move", "mirror", "duplicate"] as const }
		}
		return {
			text: `${c.description} belongs under a ${owner} node and stops working if moved elsewhere. Mirror it here instead?`,
			ops: ["mirror"] as const,
		}
	})

	const opColor = { move: "green", mirror: "blue", duplicate: "red" } as const
</script>

{#if op?.kind === "add"}
	<div class="op">
		<fieldset>
			<legend>Select node type to add:</legend>
			{#each addable as d (d.type)}
				<label class="choice">
					<input
						type="radio"
						name="add-{t.key}"
						value={d.type}
						checked={op.type === d.type}
						onchange={() => (store.op = { ...op, type: d.type })}
					/>
					<d.icon size={20} />
					{d.label}
				</label>
			{/each}
			<label class="choice">
				<input
					type="radio"
					name="add-{t.key}"
					value="custom"
					checked={op.type === "custom"}
					onchange={() => (store.op = { ...op, type: "custom" })}
				/>
				Custom
			</label>
		</fieldset>
		{#if op.type === "custom"}
			<label class="inline">
				Custom node type:
				<input type="text" bind:value={store.customNodeType} />
			</label>
		{/if}
		<div class="button-row">
			{#if op.type}
				<button
					class="btn blue"
					onclick={() =>
						void store.addNode(
							t.key,
							op.type === "custom" ? store.customNodeType : (op.type ?? "")
						)}>add</button
				>
			{/if}
			<button class="btn gray" onclick={() => store.cancelOp()}>cancel</button>
		</div>
	</div>
{:else if op?.kind === "message"}
	<div class="op">
		<label class="stack">
			Send message to users:
			<textarea
				rows="4"
				value={op.message}
				oninput={(e) => (store.op = { ...op, message: e.currentTarget.value })}></textarea>
		</label>
		<div class="button-row">
			<button class="btn blue" onclick={() => void store.sendMessage(t.key, op.message)}
				>send now</button
			>
			<button class="btn gray" onclick={() => store.cancelOp()}>cancel</button>
		</div>
	</div>
{:else if op?.kind === "delete"}
	<div class="op inline-op">
		<span>{deleteQuestion}</span>
		<div class="button-row">
			<button class="btn red" onclick={() => void store.deleteNode(t.key)}>yes</button>
			<button class="btn gray" onclick={() => store.cancelOp()}>no</button>
		</div>
	</div>
{:else if op?.kind === "paste"}
	<div class="op inline-op">
		<span>{paste.text}</span>
		<div class="button-row">
			{#each paste.ops as how (how)}
				<button class="btn {opColor[how]}" onclick={() => void store.paste(t.key, how)}
					>{how}</button
				>
			{/each}
			<button class="btn gray" onclick={() => store.cancelOp()}>cancel</button>
		</div>
	</div>
{:else}
	<div class="ops">
		<div class="buttons">
			{#if canAdd}
				<IconButton
					label="Add a node"
					onclick={() => (store.op = { kind: "add", key: t.key, type: null })}
				>
					<CirclePlus size={22} />
				</IconButton>
			{/if}
			<IconButton
				label="Message users"
				onclick={() => (store.op = { kind: "message", key: t.key, message: "" })}
			>
				<MessageSquare size={22} />
			</IconButton>
			<IconButton label="Delete" onclick={() => (store.op = { kind: "delete", key: t.key })}>
				<X size={22} />
			</IconButton>
			<IconButton label="Copy ID" onclick={() => store.copy(t.key)}>
				<Copy size={22} />
			</IconButton>
			<IconButton label="Copy link" onclick={copyLink}>
				<Link size={22} />
			</IconButton>
			<IconButton label="Paste" onclick={() => (store.op = { kind: "paste", key: t.key })}>
				<ClipboardIcon size={22} />
			</IconButton>
			<IconButton label="Raw view" pressed={viewRaw} onclick={() => store.toggleRaw(t.key)}>
				<List size={22} />
			</IconButton>
		</div>
		{#if msg}
			<div class="msg" role="status">{msg}</div>
		{/if}
	</div>
{/if}

<style>
	.ops,
	.op {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding-top: 10px;
	}

	.inline-op {
		flex-direction: row;
		flex-wrap: wrap;
		align-items: center;
	}

	.buttons {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.msg {
		white-space: pre-line;
	}

	fieldset {
		border: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	legend {
		padding: 12px;
	}

	.choice {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.inline {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}

	.stack {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	textarea {
		width: 100%;
	}
</style>
