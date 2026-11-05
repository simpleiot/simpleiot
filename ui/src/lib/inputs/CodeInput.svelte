<script lang="ts">
	// CodeInput edits a text point as a document, in a code editor with
	// line numbers and highlighting picked from a file name. The editor is
	// loaded when the input is shown, so a page without one does not carry
	// it. Every change is an edit of the point, saved with the node.
	import { untrack } from "svelte"
	import { getText, newText } from "#lib/siot/point.ts"
	import type { CodeEditor } from "./code/editor.ts"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"

	let {
		type,
		key = "0",
		label = "",
		name = "",
	}: {
		type: string
		key?: string
		label?: string
		// name is the file name the highlighting is picked from
		name?: string
	} = $props()

	const ctx = inputContext()
	const id = $props.id()

	const text = $derived(getText(ctx.node.points, type, key))

	let host: HTMLDivElement | undefined = $state()
	let editor: CodeEditor | undefined = $state.raw()
	let failed = $state("")
	// own is the last text the editor sent, so the point echoing it back
	// does not replace the document under the cursor
	let own: string | undefined

	$effect(() => {
		const el = host
		if (!el) {
			return
		}
		let live = true
		import("./code/editor.ts")
			.then(({ createCodeEditor }) => {
				if (!live) {
					return
				}
				editor = createCodeEditor(
					el,
					untrack(() => text),
					untrack(() => name),
					(t) => {
						own = t
						ctx.onEditPoints([newText(type, key, t, editTime(ctx))])
					}
				)
			})
			.catch((err: unknown) => {
				failed = err instanceof Error ? err.message : String(err)
			})
		return () => {
			live = false
			editor?.destroy()
			editor = undefined
		}
	})

	// a change from elsewhere, or the edit being cancelled, replaces the
	// document
	$effect(() => {
		const t = text
		if (editor && t !== own && t !== editor.get()) {
			own = undefined
			editor.set(t)
		}
	})

	$effect(() => {
		editor?.setName(name)
	})
</script>

<Label {label} {id} top edited={ctx.edited(type, key)}>
	{#if failed !== ""}
		<span class="error">The editor could not be loaded: {failed}</span>
	{:else}
		<div
			class="code"
			bind:this={host}
			{id}
			role="textbox"
			aria-multiline="true"
			aria-label={label}
		></div>
	{/if}
</Label>

<style>
	.code {
		flex: 1 1 300px;
		min-width: 0;
	}

	.error {
		color: var(--red);
	}
</style>
