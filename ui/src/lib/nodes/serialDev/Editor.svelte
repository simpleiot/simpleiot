<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Button from "#lib/inputs/Button.svelte"
	import ButtonAction from "#lib/inputs/ButtonAction.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import CounterWithReset from "#lib/inputs/CounterWithReset.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import OptionInput from "#lib/inputs/OptionInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { filterDeleted, getBool, getText, getValue, newValue, sorted } from "#lib/siot/point.ts"
	import { NodeType, PointType, PointValue } from "#lib/siot/schema.ts"
	import PointTable from "#lib/components/PointTable.svelte"
	import {
		descKey,
		descS,
		filterSpecialPoints,
		formatPoint,
		round,
		toMiB,
		toPercent,
		toText,
		toTime,
		toWhole,
		type Format,
	} from "#lib/components/pointFormat.ts"
	import type { EditorProps } from "../types.ts"
	import SerialDev from "#lib/icons/SerialDev.svelte"

	let props: EditorProps = $props()

	const points = $derived(props.node.points)
	const disabled = $derived(getBool(points, PointType.Disabled))
	const connected = $derived(getBool(points, PointType.Connected))

	const log = $derived(getText(points, PointType.Log))
	// an empty protocol means binary, so nodes created before shell mode
	// existed still render correctly
	const isShell = $derived(getText(points, PointType.Protocol) === PointValue.ProtocolShell)
	const rateS = $derived(String(Math.round(getValue(points, PointType.Rate))))
	const rateHRS = $derived(String(Math.round(getValue(points, PointType.RateHR))))
	const download = $derived(getText(points, PointType.Download))
	const downloading = $derived(download.length > 0)
	const progress = $derived(Math.round(getValue(points, PointType.Progress)))
	const errorS = $derived(getText(points, PointType.Error))
	const fwVersion = $derived(getText(points, PointType.FwVersion))
	const fwStaged = $derived(getText(points, PointType.FwStagedVersion))
	const fwPending = $derived(getBool(points, PointType.FwPending))
	// the MCU reports "" or "none" when no image is staged
	const fwStagedS = $derived(fwStaged === "" || fwStaged === "none" ? "none" : fwStaged)
	const fwUploading = $derived(isShell && progress > 0 && progress < 100)

	const files = $derived(props.children.filter((n) => n.node.type === NodeType.File))
	const fileOptions = $derived(
		files.map((n): [string, string] => {
			const filename = getText(n.node.points, PointType.Name)
			return [filename, `${getText(n.node.points, PointType.Description)}: ${filename}`]
		})
	)

	// sent right away rather than staged for Save, since it is an action,
	// not a setting
	function applyFirmware() {
		props.onSendPoints([newValue(PointType.FwApply, "0", 1, new Date(props.now).toISOString())])
	}

	const formats: Record<string, Format> = {
		metricAppAlloc: { desc: descS("App Memory Alloc"), vf: toMiB },
		metricAppNumGoroutine: { desc: descS("App Goroutine Count"), vf: toWhole },
		metricProcCPUPercent: { desc: descS("Proc CPU %"), vf: toPercent },
		metricProcMemPercent: { desc: descS("Proc Mem %"), vf: toPercent },
		metricProcMemRSS: { desc: descS("Proc Mem RSS"), vf: toMiB },
		host: { desc: descKey("Host"), vf: toText },
		hostBootTime: { desc: descS("Host Boot Time"), vf: toTime },
		metricSysCPUPercent: { desc: descS("Sys CPU %"), vf: toPercent },
		metricSysDiskUsedPercent: { desc: descKey("Disk Used %"), vf: toPercent },
		metricSysLoad: { desc: descKey("Load"), vf: (p) => round(2, p.value) },
		metricSysMemUsedPercent: { desc: descS("Memory used %"), vf: toPercent },
		metricSysMem: { desc: descKey("Memory"), vf: toMiB },
		metricSysNetBytesRecv: { desc: descKey("Net RX"), vf: toWhole },
		metricSysNetBytesSent: { desc: descKey("Net TX"), vf: toWhole },
		metricSysUptime: { desc: descKey("Uptime"), vf: toWhole },
	}

	const rows = $derived(
		filterDeleted(filterSpecialPoints(sorted(points))).map((p) => formatPoint(formats, p))
	)
</script>

<NodeEditor
	{props}
	icon={SerialDev}
	labelWidth={180}
	tone={disabled || !connected ? "muted" : undefined}
>
	{#snippet header()}
		{#if !connected}<span>(not connected)</span>{/if}
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.Port} label="Port" placeholder="/dev/ttyUSB0" />
	<TextInput type={PointType.Baud} label="Baud" placeholder="9600" />
	<OptionInput
		type={PointType.Protocol}
		label="Protocol"
		options={[
			[PointValue.ProtocolBinary, "Binary (COBS)"],
			[PointValue.ProtocolShell, "Zephyr shell"],
		]}
	/>
	<NumberInput type={PointType.MaxMessageLength} label="Max Msg Len" />
	{#if isShell}
		<NumberInput type={PointType.Timeout} label="Timeout (sec)" />
		<CheckboxInput type={PointType.LogConsole} label="Log console output" />
	{:else}
		<TextInput type={PointType.HRDest} label="HR Dest Node" />
	{/if}
	<CheckboxInput type={PointType.SyncParent} label="Sync parent node" />
	<NumberInput type={PointType.Debug} label="Debug level (0-9)" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<div class="rule"></div>
	<CounterWithReset
		type={PointType.ErrorCount}
		resetType={PointType.ErrorCountReset}
		label="Error Count"
	/>
	<CounterWithReset type={PointType.Rx} resetType={PointType.RxReset} label="Rx count" />
	<CounterWithReset type={PointType.Tx} resetType={PointType.TxReset} label="Tx count" />
	<!-- high rate and file transfer have no shell equivalent -->
	{#if !isShell}
		<CounterWithReset
			type={PointType.ErrorCountHR}
			resetType={PointType.ErrorCountResetHR}
			label="HR err count"
		/>
		<CounterWithReset type={PointType.HrRx} resetType={PointType.HrRxReset} label="HR Rx count" />
	{/if}
	<div class="status">{`  Rate (pts/sec): ${rateS}`}</div>
	{#if !isShell}
		<div class="status">{`  Rate HR (pkts/sec): ${rateHRS}`}</div>
		<div class="status">{`  Last log: ${log}`}</div>
	{/if}
	{#if errorS !== ""}
		<p class="error">Error: {errorS}</p>
	{/if}
	{#if isShell}
		<div class="rule"></div>
		<TextInput
			type={PointType.FirmwarePath}
			label="Firmware image"
			placeholder="/lib/firmware/mcu.signed.bin"
		/>
		<CheckboxInput type={PointType.FwAutoApply} label="Auto apply firmware" />
		{#if fwVersion !== ""}
			<div class="status">{`  Running firmware: ${fwVersion}`}</div>
			<div class="status">
				{`  Staged firmware: ${fwStagedS}${fwPending ? " (pending)" : ""}`}
			</div>
		{/if}
	{/if}
	{#if fwUploading}
		<div class="status">{`  Firmware upload: ${progress} %`}</div>
	{/if}
	{#if isShell && fwPending}
		<div class="row">
			<Button onclick={applyFirmware}>Apply firmware now (restarts this system)</Button>
		</div>
	{/if}
	{#if !isShell && files.length > 0}
		<div class="rule"></div>
		{#if !downloading}
			<OptionInput type={PointType.Download} label="Download file" options={fileOptions} />
		{/if}
	{/if}
	{#if downloading}
		<div>Downloading: {download}</div>
		<div>Progress: {progress} %</div>
		<div class="row">
			<ButtonAction type={PointType.Download} text="" label="Cancel download" color="red" />
		</div>
	{/if}
	<div class="rule"></div>
	<PointTable {rows} label="Point" />
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>

<style>
	.rule {
		border-bottom: 2px solid var(--muted);
	}

	.status {
		white-space: pre;
	}

	.error {
		margin: 0;
		max-width: 500px;
		padding: 8px;
		border: 1px solid var(--red);
		border-radius: 4px;
		overflow-wrap: anywhere;
	}

	.row {
		display: flex;
	}
</style>
