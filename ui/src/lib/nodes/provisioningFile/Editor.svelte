<script lang="ts">
	// A provisioning file on disk, and what provisioning last did with it.
	// This is a status rather than a setting: everything here is recorded by
	// the server, so nothing on it is editable.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import { get, getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { toDateTimeString } from "#lib/util/iso8601.ts"
	import type { EditorProps } from "../types.ts"
	import FileIcon from "@lucide/svelte/icons/file"

	let props: EditorProps = $props()

	const err = $derived(getText(props.node.points, PointType.Error))

	// The hash is what provisioning compares to decide whether there is
	// anything to do, and the timestamp on that point is when it last did it.
	const hashPoint = $derived(get(props.node.points, PointType.Hash))
	const applied = $derived(hashPoint ? toDateTimeString(hashPoint.time) : "never")

	// shortHash keeps a checksum readable. The whole thing says nothing more
	// to a person than its first few characters do.
	function shortHash(hash: string): string {
		if (hash.length > 12) {
			return hash.slice(0, 12)
		}
		return hash === "" ? "unknown" : hash
	}
</script>

<NodeEditor {props} icon={FileIcon}>
	{#snippet header()}
		<!-- says at a glance whether a file applied, without expanding it -->
		{#if err !== ""}
			<span class="error">{err}</span>
		{/if}
	{/snippet}
	<div class="status">
		<div>Applied: {applied}</div>
		<div>Contents: {shortHash(hashPoint?.text ?? "")}</div>
	</div>
</NodeEditor>

<style>
	.error {
		color: var(--red);
	}

	.status {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding-left: 70px;
	}
</style>
