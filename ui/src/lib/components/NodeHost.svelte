<script lang="ts">
	// NodeHost edits one node off the tree: the node's registry editor with
	// its own pending edit, save and discard, and an optional delete. The
	// node must be among the store's live nodes, which the page arranges.
	import { registry } from "#lib/nodes/index.ts"
	import type { EditorProps } from "#lib/nodes/types.ts"
	import type { Node } from "#lib/siot/node.ts"
	import { clearText, updatePoints, type Point } from "#lib/siot/point.ts"
	import { store } from "#lib/siot/store.ts"
	import Raw from "./Raw.svelte"

	let {
		id,
		parent,
		onDelete,
		onCopy,
		onDuplicate,
	}: {
		id: string
		// parent is the node the edited one sits under, when the page knows
		parent?: Node
		// onDelete, onCopy, and onDuplicate, when given, offer those buttons
		onDelete?: () => void
		onCopy?: () => void
		onDuplicate?: () => void
	} = $props()

	let pending = $state<Point[]>([])
	let scratch = $state("")

	// a file node's editor uploads through a hidden input, as in the tree
	let fileInput: HTMLInputElement | undefined = $state()
	let uploadBinary = false
	function onFile(e: Event & { currentTarget: HTMLInputElement }) {
		const file = e.currentTarget.files?.[0]
		e.currentTarget.value = ""
		if (file) {
			void store.uploadFile(id, uploadBinary, file)
		}
	}

	const live = $derived(store.liveNodes.get(id))
	const node = $derived(
		live && pending.length > 0 ? { ...live, points: updatePoints(live.points, pending) } : live
	)
	const Editor = $derived(live ? (registry.editorFor(live.type) ?? Raw) : Raw)

	// a different node starts with no edit
	let lastId = ""
	$effect(() => {
		if (id !== lastId) {
			lastId = id
			pending = []
			scratch = ""
		}
	})

	const editorProps = $derived.by((): EditorProps | null =>
		node
			? {
					node,
					typeLabel: registry.get(node.type)?.label ?? node.type,
					parent,
					children: [],
					now: store.now,
					zone: store.zone,
					expDetail: true,
					modified: pending.length > 0,
					pending,
					copy: store.clipboard,
					scratch,
					generatedToken: null,
					findNode: (nodeId: string) => store.findNode(nodeId) ?? store.liveNodes.get(nodeId),
					searchNodes: (query: string, type?: string) => store.searchNodes(query, 20, type),
					resolveNode: (nodeId: string) => store.resolveNode(nodeId),
					onEditPoints: (points: Point[]) => {
						pending = updatePoints(pending, points)
						scratch = ""
					},
					onSendPoints: (points: Point[]) => void store.sendPoints(id, points),
					onEditScratch: (text: string) => (scratch = text),
					onUploadFile: (binary: boolean) => {
						uploadBinary = binary
						fileInput?.click()
					},
					onGenerateKey: () => undefined,
				}
			: null
	)

	function save() {
		const points = clearText(pending)
		pending = []
		if (points.length > 0) {
			void store.sendPoints(id, points)
		}
	}

	function onkeyup(e: KeyboardEvent) {
		const target = e.target as HTMLElement
		if (target.tagName === "TEXTAREA" || target.tagName === "BUTTON") {
			return
		}
		if (e.key === "Enter" && pending.length > 0) {
			save()
		} else if (e.key === "Escape" && pending.length > 0) {
			pending = []
		}
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="host" class:modified={pending.length > 0} {onkeyup}>
	{#if editorProps}
		<Editor {...editorProps} />
		<div class="bar">
			{#if pending.length > 0}
				<span class="unsaved">Unsaved changes</span>
				<button class="btn blue" onclick={save}>save</button>
				<button class="btn gray" onclick={() => (pending = [])}>discard</button>
			{/if}
			{#if onDuplicate}
				<button class="btn green" onclick={onDuplicate}>duplicate</button>
			{/if}
			{#if onCopy}
				<button class="btn gray" onclick={onCopy}>copy</button>
			{/if}
			{#if onDelete}
				<button class="btn red" onclick={onDelete}>delete</button>
			{/if}
		</div>
	{:else}
		<p class="muted">Node {id} is not loaded.</p>
	{/if}
	<input
		type="file"
		class="visually-hidden"
		tabindex="-1"
		bind:this={fileInput}
		onchange={onFile}
	/>
</div>

<style>
	.host {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 12px;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--detail-bg);
	}

	.modified {
		border-color: var(--blue);
	}

	.bar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}

	.unsaved {
		color: var(--blue);
		font-weight: 600;
	}

	.muted {
		margin: 0;
		color: var(--muted);
	}
</style>
