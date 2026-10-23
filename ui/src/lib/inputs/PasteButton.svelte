<script lang="ts">
	// PasteButton sets a text point to a value, such as the ID of the node
	// on the clipboard.
	import ClipboardIcon from "@lucide/svelte/icons/clipboard"
	import type { Snippet } from "svelte"
	import IconButton from "#lib/components/IconButton.svelte"
	import { newText } from "#lib/siot/point.ts"
	import { editTime, inputContext } from "./context.ts"
	import Indent from "./Indent.svelte"

	let { type, value, children }: { type: string; value: string; children: Snippet } = $props()

	const ctx = inputContext()
</script>

<Indent>
	<div class="paste">
		<IconButton
			label="Paste"
			onclick={() => {
				ctx.onEditPoints([newText(type, "0", value, editTime(ctx))])
			}}
		>
			<ClipboardIcon size={22} />
		</IconButton>
		{@render children()}
	</div>
</Indent>

<style>
	.paste {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}
</style>
