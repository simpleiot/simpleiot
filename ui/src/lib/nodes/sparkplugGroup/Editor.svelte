<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import RssIcon from "@lucide/svelte/icons/rss"

	let props: EditorProps = $props()

	const group = $derived(getText(props.node.points, PointType.ID))
</script>

<NodeEditor {props} icon={RssIcon}>
	<div class="note">Sparkplug group: {group}</div>
	<TextInput type={PointType.Description} label="Description" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.note {
		padding-left: 70px;
	}
</style>
