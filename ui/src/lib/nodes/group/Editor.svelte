<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Users from "@lucide/svelte/icons/users"

	let props: EditorProps = $props()
</script>

<NodeEditor {props} icon={Users} labelWidth={100}>
	<TextInput type={PointType.Description} label="Description" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>
