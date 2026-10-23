<script lang="ts">
	// One editor serves both actions run while the rule is active and those
	// run while it is inactive.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OnOffInput from "#lib/inputs/OnOffInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import PasteButton from "#lib/inputs/PasteButton.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getBestDesc, getBool, getText } from "#lib/siot/point.ts"
	import { NodeType, PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import TrendingDown from "@lucide/svelte/icons/trending-down"
	import TrendingUp from "@lucide/svelte/icons/trending-up"

	let props: EditorProps = $props()

	const points = $derived(props.node.points)
	const icon = $derived(props.node.type === NodeType.Action ? TrendingUp : TrendingDown)
	const active = $derived(getBool(points, PointType.Active))
	const error = $derived(getText(points, PointType.Error))
	const disabled = $derived(getBool(points, PointType.Disabled))

	const actionType = $derived(getText(points, PointType.Action))
	const actionSetValue = $derived(actionType === PointValue.SetValue)
	const actionPlayAudio = $derived(actionType === PointValue.PlayAudio)
	const actionNotify = $derived(actionType === PointValue.Notify)
	const valueType = $derived(getText(points, PointType.ValueType))
	const nodeId = $derived(getText(points, PointType.NodeID))
	const target = $derived(nodeId === "" ? undefined : props.findNode(nodeId))
</script>

<NodeEditor
	{props}
	{icon}
	tone={disabled ? "muted" : error !== "" ? "error" : undefined}
	highlight={active}
>
	<TextInput type={PointType.Description} label="Description" />
	<OptionInput
		type={PointType.Action}
		label="Action"
		options={[
			[PointValue.Notify, "notify"],
			[PointValue.SetValue, "set node value"],
			[PointValue.PlayAudio, "play audio"],
		]}
	/>
	{#if actionNotify}
		<NumberInput type={PointType.RepeatInterval} label="Repeat interval (m)" />
	{/if}
	{#if actionSetValue}
		<OptionInput
			type={PointType.PointType}
			label="Point Type"
			options={[
				[PointType.Value, "value"],
				[PointType.ValueSet, "set value (use for remote devices)"],
				[PointType.LightSet, "set light state"],
				[PointType.SwitchSet, "set switch state"],
			]}
		/>
		<TextInput type={PointType.PointKey} label="Point Key" />
		<TextInput type={PointType.NodeID} label="Node ID" />
	{/if}
	{#if nodeId !== ""}
		<div class="node-desc">
			{#if target}
				<span class="found">({getBestDesc(target.points)})</span>
			{:else}
				<span class="not-found">(node not found)</span>
			{/if}
		</div>
	{/if}
	{#if actionSetValue && props.copy && nodeId !== props.copy.id}
		<PasteButton type={PointType.NodeID} value={props.copy.id}>
			<span>paste ID for node:</span>
			<span class="copy-desc">{props.copy.description}</span>
		</PasteButton>
	{/if}
	{#if actionSetValue}
		<OptionInput
			type={PointType.ValueType}
			label="Point Value Type"
			options={[
				[PointValue.Number, "number"],
				[PointValue.OnOff, "on/off"],
				[PointValue.Text, "text"],
			]}
		/>
		{#if valueType === PointValue.Number}
			<NumberInput type={PointType.Value} label="Value" />
		{:else if valueType === PointValue.OnOff}
			<OnOffInput type={PointType.Value} setType={PointType.Value} label="Value" />
		{:else if valueType === PointValue.Text}
			<TextInput type={PointType.ValueText} label="Value" />
		{/if}
	{/if}
	{#if actionPlayAudio}
		<TextInput type={PointType.Device} label="Device" />
		<NumberInput type={PointType.Channel} label="Channel" />
		<TextInput
			type={PointType.FilePath}
			label="Wav file path"
			placeholder="/absolute/path/to/sound.wav"
		/>
	{/if}
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<div class="error-text">{error}</div>
</NodeEditor>

<style>
	.node-desc {
		font-style: italic;
		padding-left: 170px;
	}

	.found,
	.copy-desc {
		background: var(--badge-bg);
		color: var(--badge-fg);
	}

	.copy-desc {
		font-style: italic;
	}

	.not-found {
		background: var(--orange);
		color: #000000;
	}

	.error-text {
		color: var(--red);
	}

	@media (max-width: 600px) {
		.node-desc {
			padding-left: 0;
		}
	}
</style>
