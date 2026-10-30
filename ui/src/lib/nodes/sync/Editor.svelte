<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import { onlineStatus } from "#lib/siot/node.ts"
	import { get, getText } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw"
	import Cloud from "@lucide/svelte/icons/cloud"
	import CloudOff from "@lucide/svelte/icons/cloud-off"

	let props: EditorProps = $props()

	const error = $derived(getText(props.node.points, PointType.Error))
	const authToken = $derived(getText(props.node.points, PointType.AuthToken))
	const pubKey = $derived(getText(props.node.points, PointType.PubKey))
	// the sync client sets online when the upstream connection comes up or
	// drops
	const online = $derived(get(props.node.points, PointType.Online))
	const status = $derived(onlineStatus(props.node.points, props.now))
</script>

<NodeEditor {props} icon={RefreshCwIcon} labelWidth={180}>
	{#snippet header()}
		{#if online?.value === 0}
			<CloudOff size={20} />
		{:else if online?.value === 1}
			<Cloud size={20} />
		{/if}
		{#if error !== ""}
			<span class="error">({error})</span>
		{/if}
	{/snippet}
	{#if status !== ""}
		<Value label="Status">{status}</Value>
	{/if}
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.URI} label="URI" placeholder="nats://myserver:4222, ws://myserver" />
	<TextInput type={PointType.AuthToken} label="Auth Token" />
	{#if authToken === ""}
		<!-- the key this instance connects with when no token is set, which
		is what an upstream enrolls -->
		<div class="device-key">
			<span>Device key:</span>
			<span class="key">{pubKey === "" ? "(not yet reported)" : pubKey}</span>
		</div>
		<TextInput
			type={PointType.EnrollToken}
			label="Enroll Token"
			placeholder="optional, from the upstream"
		/>
	{/if}
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<CounterWithReset
		type={PointType.SyncCount}
		resetType={PointType.SyncCountReset}
		label="Sync Count"
	/>
	<CounterWithReset type={PointType.TxBytes} resetType={PointType.TxBytesReset} label="Tx Bytes" />
	<CounterWithReset type={PointType.RxBytes} resetType={PointType.RxBytesReset} label="Rx Bytes" />
	<NumberInput type={PointType.BytesReportPeriod} label="Report Period (s)" />
</NodeEditor>

<style>
	.error {
		color: var(--red);
	}

	.device-key {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px;
		padding: 6px 0;
	}

	.key {
		font-family: var(--mono);
		font-size: 14px;
		overflow-wrap: anywhere;
	}
</style>
