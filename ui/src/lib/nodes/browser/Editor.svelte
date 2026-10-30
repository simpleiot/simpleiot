<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import GlobeIcon from "@lucide/svelte/icons/globe"

	let props: EditorProps = $props()
</script>

<NodeEditor {props} icon={GlobeIcon}>
	<TextInput type={PointType.URL} label="URL" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<NumberInput type={PointType.Rotate} label="Rotate" />
	<CheckboxInput type={PointType.KeyboardScale} label="Keyboard Scale" />
	<CheckboxInput type={PointType.Fullscreen} label="Fullscreen" />
	<CheckboxInput type={PointType.DefaultDialogs} label="Default Dialogs" />
	<TextInput type={PointType.DialogColor} label="Dialog Color" />
	<CheckboxInput type={PointType.TouchQuirk} label="Touch Quirk" />
	<NumberInput type={PointType.RetryInterval} label="Retry Interval" />
	<TextInput type={PointType.ExceptionURL} label="Exception URL" />
	<CheckboxInput type={PointType.IgnoreCertErr} label="Ignore Cert Error" />
	<CheckboxInput type={PointType.DisableSandbox} label="Disable Sandbox" />
	<TextInput type={PointType.DebugPort} label="Debug Port" />
	<TextInput type={PointType.ScreenResolution} label="Screen Resolution" />
	<TextInput type={PointType.DisplayCard} label="Display Card" />
</NodeEditor>
