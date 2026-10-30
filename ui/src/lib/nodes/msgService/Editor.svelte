<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import SendIcon from "@lucide/svelte/icons/send"

	let props: EditorProps = $props()

	const service = $derived(getText(props.node.points, PointType.Service))
	const isTwilio = $derived(service === PointValue.Twilio)
	const isSMTP = $derived(service === PointValue.SMTP)
	const isNtfy = $derived(service === PointValue.Ntfy)
</script>

<NodeEditor {props} icon={SendIcon}>
	<TextInput type={PointType.Description} label="Description" />
	<OptionInput
		type={PointType.Service}
		label="Service"
		options={[
			[PointValue.Twilio, "Twilio SMS"],
			[PointValue.SMTP, "Email (SMTP)"],
			[PointValue.Ntfy, "ntfy push"],
		]}
	/>
	{#if isTwilio}
		<TextInput type={PointType.SID} label="SID" />
		<TextInput type={PointType.AuthToken} label="Auth Token" />
		<TextInput type={PointType.From} label="From" placeholder="+15555555555" />
	{/if}
	{#if isSMTP}
		<TextInput type={PointType.URL} label="Server" placeholder="smtp.example.com:587" />
		<TextInput type={PointType.Username} label="Username" />
		<TextInput type={PointType.AuthToken} label="Password" />
		<TextInput type={PointType.From} label="From" placeholder="siot@example.com" />
	{/if}
	{#if isNtfy}
		<TextInput type={PointType.URL} label="Server" placeholder="https://ntfy.sh" />
		<TextInput type={PointType.Topic} label="Topic" />
		<TextInput type={PointType.AuthToken} label="Access Token" placeholder="(optional)" />
	{/if}
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>
