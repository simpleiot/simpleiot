<script lang="ts">
	// NodeEditor is the frame of a node editor: the header line, then,
	// while the node's detail is open, the editor's inputs. It gives the
	// inputs their context.
	import type { Snippet } from "svelte"
	import { setInputContext } from "#lib/inputs/context.ts"
	import type { Icon } from "#lib/nodes/registry.ts"
	import type { EditorProps } from "#lib/nodes/types.ts"
	import EditorHeader from "./EditorHeader.svelte"

	let {
		props,
		icon,
		labelWidth = 150,
		description,
		tone,
		highlight = false,
		showDisabled = true,
		lead,
		header,
		summary,
		children,
	}: {
		props: EditorProps
		icon: Icon
		labelWidth?: number
		description?: string
		// tone, highlight, showDisabled, and lead are passed to EditorHeader
		tone?: "muted" | "error" | undefined
		highlight?: boolean
		showDisabled?: boolean
		lead?: Snippet
		// header follows the description on the header line
		header?: Snippet
		// summary shows below the header whether or not the detail is open
		summary?: Snippet
		// children are the inputs, shown while the detail is open
		children?: Snippet
	} = $props()

	setInputContext({
		get node() {
			return props.node
		},
		get now() {
			return props.now
		},
		get labelWidth() {
			return labelWidth
		},
		get scratch() {
			return props.scratch
		},
		onEditPoints: (points) => {
			props.onEditPoints(points)
		},
		onEditScratch: (text) => {
			props.onEditScratch(text)
		},
	})
</script>

<div class="editor">
	<EditorHeader node={props.node} {icon} {description} {tone} {highlight} {showDisabled} {lead}
		>{@render header?.()}</EditorHeader
	>
	{@render summary?.()}
	{#if props.expDetail && children}
		<div class="detail">
			{@render children()}
		</div>
	{/if}
</div>

<style>
	.editor {
		border-top: 2px solid var(--fg);
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.detail {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding-bottom: 6px;
	}
</style>
