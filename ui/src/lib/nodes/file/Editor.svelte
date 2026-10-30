<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Button from "#lib/inputs/Button.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { get, getBool, getText, getValue } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { toDateTimeString } from "#lib/util/iso8601.ts"
	import type { EditorProps } from "../types.ts"
	import FileIcon from "@lucide/svelte/icons/file"

	let props: EditorProps = $props()

	const points = $derived(props.node.points)
	const desc = $derived(getText(points, PointType.Description))
	const name = $derived(getText(points, PointType.Name))
	const binary = $derived(getBool(points, PointType.Binary))
	const size = $derived(getValue(points, PointType.Size))
	const data = $derived(getText(points, PointType.Data))
	const hash = $derived(getText(points, PointType.Hash))
	const created = $derived(getValue(points, PointType.Created))
	const provisionHash = $derived(getText(points, PointType.ProvisionHash))
	const provisionError = $derived(getText(points, PointType.Error))
	const provisionApplied = $derived.by(() => {
		const p = get(points, PointType.ProvisionHash)
		return p ? toDateTimeString(p.time) : "never"
	})

	// formatCreated reads the point provisioning orders uploads by. A file
	// node that predates the point has none, which is worth saying rather
	// than showing an epoch date.
	function formatCreated(t: number): string {
		return t <= 0 ? "unknown" : toDateTimeString(Math.round(t * 1000))
	}
</script>

<NodeEditor {props} icon={FileIcon} description={desc + " (" + name + ")"}>
	<TextInput type={PointType.Description} label="Description" />
	<CheckboxInput type={PointType.Binary} label="Binary" />
	<div class="status">&nbsp;</div>
	<div class="status">{"     File name: " + name}</div>
	<div class="status">{"     File size: " + String(size) + " bytes"}</div>
	<div class="status">{"     File md5: " + hash}</div>
	<div class="status">{"     Added: " + formatCreated(created)}</div>
	<div class="status">{"     Stored data len: " + String(data.length) + " bytes"}</div>
	<div class="button-row">
		<Button color="blue" onclick={() => props.onUploadFile(binary)}>Upload new file</Button>
	</div>
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<!-- what provisioning did with this file, shown only when provisioning
	has used it, so a file node holding a CAN database or a serial
	configuration is unaffected -->
	{#if provisionHash !== ""}
		<div class="status">&nbsp;</div>
		<div class="status">{"     Provisioned: " + provisionApplied}</div>
		<div class="status">{"     Applied contents: " + provisionHash.slice(0, 12)}</div>
		{#if provisionError !== ""}
			<div class="status error">{"     Error: " + provisionError}</div>
		{/if}
	{/if}
</NodeEditor>

<style>
	.status {
		white-space: pre-wrap;
	}

	.error {
		color: var(--red);
	}
</style>
