<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Shelly from "#lib/icons/Shelly.svelte"

	let props: EditorProps = $props()
</script>

<NodeEditor {props} icon={Shelly} labelWidth={180}>
	<TextInput type={PointType.Description} label="Description" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
</NodeEditor>
