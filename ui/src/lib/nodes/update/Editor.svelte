<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Button from "#lib/inputs/Button.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import {
		filterDeleted,
		get,
		getAll,
		getText,
		newText,
		newValue,
		sorted,
	} from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw"

	let props: EditorProps = $props()

	const points = $derived(props.node.points)
	const osDownloaded = $derived(getText(points, PointType.OSDownloaded))
	const downloadOS = $derived(getText(points, PointType.DownloadOS))
	const error = $derived(getText(points, PointType.Error))
	const osUpdates = $derived(sorted(filterDeleted(getAll(points, PointType.OSUpdate))))

	function version(type: string, label: string): string {
		const p = get(points, type)
		return p ? label + ": " + p.text : ""
	}

	const versionHW = $derived(version(PointType.VersionHW, "HW"))
	const versionOS = $derived(version(PointType.VersionOS, "OS"))
	const versionApp = $derived(version(PointType.VersionApp, "App"))

	const time = () => new Date(props.now).toISOString()

	// press sets a point the client acts on and then clears
	function press(type: string) {
		props.onEditPoints([newValue(type, "0", 1, time())])
	}

	function install(text: string) {
		props.onEditPoints([newText(PointType.DownloadOS, "0", text, time())])
	}
</script>

<NodeEditor {props} icon={RefreshCwIcon} labelWidth={165}>
	<TextInput type={PointType.Description} label="Description" />
	<TextInput type={PointType.URI} label="Update server" placeholder="http://..." />
	<TextInput type={PointType.Prefix} label="Prefix" />
	<TextInput type={PointType.Directory} label="Dest dir" />
	<NumberInput type={PointType.PollPeriod} label="Chk interval (min)" />
	<CheckboxInput type={PointType.AutoDownload} label="Auto download" />
	<CheckboxInput type={PointType.AutoReboot} label="Auto reboot/install" />
	{#if versionHW !== "" || versionOS !== "" || versionApp !== ""}
		<div>{"Current version: " + versionHW + " " + versionOS + " " + versionApp}</div>
	{/if}
	{#if osDownloaded !== ""}
		<div class="info">OS downloaded, reboot to install: {osDownloaded}</div>
	{:else if downloadOS !== ""}
		<div class="info">Downloading OS version: {downloadOS}</div>
	{:else}
		<div>
			<div class="updates-label">OS Updates:</div>
			<table>
				<tbody>
					{#each osUpdates as p (p.key)}
						<tr>
							<td>{p.text}</td>
							<td class="install">
								<Button color="blue" onclick={() => install(p.text)}>install</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
	<div class="error">{error}</div>
	<div class="button-row">
		<Button color="blue" onclick={() => press(PointType.Refresh)}>Refresh</Button>
		{#if osDownloaded !== ""}
			<!-- the kit's Button has no orange; this is the same .btn in orange -->
			<button type="button" class="btn orange" onclick={() => press(PointType.DiscardDownload)}
				>Discard</button
			>
			<Button color="red" onclick={() => press(PointType.Reboot)}>Reboot</Button>
		{/if}
	</div>
</NodeEditor>

<style>
	.info {
		color: var(--blue);
	}

	.error {
		color: var(--red);
	}

	.updates-label {
		padding: 0 20px;
	}

	table {
		margin-left: 70px;
		border-collapse: collapse;
	}

	td {
		vertical-align: middle;
	}

	.install {
		padding: 2px;
	}

	.orange {
		--c: var(--orange);
	}
</style>
