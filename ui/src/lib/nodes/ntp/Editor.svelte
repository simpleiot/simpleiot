<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import ListInput from "#lib/inputs/ListInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import ClockIcon from "@lucide/svelte/icons/clock"

	let props: EditorProps = $props()
</script>

<NodeEditor {props} icon={ClockIcon}>
	<TextInput type={PointType.Description} label="Description" />
	<ListInput type={PointType.Server} label="Servers" buttonLabel="Add Server" />
	<ListInput
		type={PointType.FallbackServer}
		label="Fallback Servers"
		buttonLabel="Add Fallback Server"
	/>
</NodeEditor>
