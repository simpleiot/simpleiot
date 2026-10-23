<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getBool, getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import ListIcon from "@lucide/svelte/icons/list"

	let props: EditorProps = $props()

	const active = $derived(getBool(props.node.points, PointType.Active))
	const error = $derived(getText(props.node.points, PointType.Error))
	const disabled = $derived(getBool(props.node.points, PointType.Disabled))
</script>

<!-- the header line is gray while disabled and red on an error, and the
description is highlighted while the rule is active -->
<NodeEditor
	{props}
	icon={ListIcon}
	labelWidth={100}
	tone={disabled ? "muted" : error !== "" ? "error" : undefined}
	highlight={active}
>
	<TextInput type={PointType.Description} label="Description" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<div class="error-text">{error}</div>
</NodeEditor>

<style>
	.error-text {
		color: var(--red);
	}
</style>
