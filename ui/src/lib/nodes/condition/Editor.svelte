<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import NodeRefInput from "#lib/inputs/NodeRefInput.svelte"
	import OnOffInput from "#lib/inputs/OnOffInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import ScheduleInput from "#lib/inputs/ScheduleInput.svelte"
	import SelectInput from "#lib/inputs/SelectInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { pointKeys, pointTypes, type Node } from "#lib/siot/node.ts"
	import { getBool, getText } from "#lib/siot/point.ts"
	import { PointType, PointValue } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Check from "@lucide/svelte/icons/check"

	let props: EditorProps = $props()

	const points = $derived(props.node.points)
	const active = $derived(getBool(points, PointType.Active))
	const error = $derived(getText(points, PointType.Error))
	const disabled = $derived(getBool(points, PointType.Disabled))
	const conditionType = $derived(getText(points, PointType.ConditionType))

	const conditionValueType = $derived(getText(points, PointType.ValueType))
	const nodeId = $derived(getText(points, PointType.NodeID))
	const pointType = $derived(getText(points, PointType.PointType))

	// the target node, from the page or fetched, for its point types and keys
	let resolved = $state<{ id: string; node: Node | undefined } | null>(null)
	const onPage = $derived(nodeId === "" ? undefined : props.findNode(nodeId))
	$effect(() => {
		const id = nodeId
		if (id === "" || onPage || resolved?.id === id) {
			return
		}
		void props.resolveNode(id).then((node) => {
			resolved = { id, node }
		})
	})
	const target = $derived(onPage ?? (resolved?.id === nodeId ? resolved.node : undefined))

	const operators = $derived.by((): [string, string][] => {
		switch (conditionValueType) {
			case PointValue.Number:
				return [
					[PointValue.GreaterThan, ">"],
					[PointValue.LessThan, "<"],
					[PointValue.Equal, "="],
					[PointValue.NotEqual, "!="],
				]
			case PointValue.Text:
				return [
					[PointValue.Equal, "="],
					[PointValue.NotEqual, "!="],
					[PointValue.Contains, "contains"],
				]
			default:
				return []
		}
	})
</script>

<NodeEditor
	{props}
	icon={Check}
	tone={disabled ? "muted" : error !== "" ? "error" : undefined}
	highlight={active}
>
	<TextInput type={PointType.Description} label="Description" />
	<OptionInput
		type={PointType.ConditionType}
		label="Type"
		options={[
			[PointValue.PointValue, "point value"],
			[PointValue.Schedule, "schedule"],
		]}
	/>
	{#if conditionType === PointValue.PointValue}
		<NodeRefInput
			type={PointType.NodeID}
			label="Node"
			search={props.searchNodes}
			resolve={props.resolveNode}
			findNode={props.findNode}
			suggest={props.copy}
		/>
		<SelectInput
			type={PointType.PointType}
			label="Point Type"
			options={[
				[PointType.Value, "value"],
				[PointType.ValueSet, "set value"],
				[PointType.ErrorCount, "error count"],
				[PointType.Online, "online"],
				[PointType.Active, "active"],
			]}
			extra={target ? pointTypes(target) : []}
		/>
		<TextInput
			type={PointType.PointKey}
			label="Point Key"
			suggestions={target ? pointKeys(target, pointType) : []}
		/>
		<OptionInput
			type={PointType.ValueType}
			label="Point Value Type"
			options={[
				[PointValue.Number, "number"],
				[PointValue.OnOff, "on/off"],
				[PointValue.Text, "text"],
			]}
		/>
		{#if conditionValueType !== PointValue.OnOff}
			<OptionInput type={PointType.Operator} label="Operator" options={operators} />
		{/if}
		{#if conditionValueType === PointValue.Number}
			<NumberInput type={PointType.Value} label="Point Value" />
		{:else if conditionValueType === PointValue.OnOff}
			<OnOffInput type={PointType.Value} setType={PointType.Value} label="Point Value" />
		{:else if conditionValueType === PointValue.Text}
			<TextInput type={PointType.ValueText} label="Point Value" />
		{/if}
		<NumberInput type={PointType.MinActive} label="Min active time (m)" />
		<NumberInput type={PointType.MinInactive} label="Min inactive time (m)" />
	{:else if conditionType === PointValue.Schedule}
		<ScheduleInput />
	{:else}
		<div class="error-text">Please select condition type</div>
	{/if}
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
	<div class="error-text">{error}</div>
</NodeEditor>

<style>
	.error-text {
		color: var(--red);
	}
</style>
