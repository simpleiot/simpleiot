<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import NetworkIcon from "@lucide/svelte/icons/network"

	let props: EditorProps = $props()
</script>

<NodeEditor {props} icon={NetworkIcon}>
	<TextInput type={PointType.Description} label="Description" />
</NodeEditor>
