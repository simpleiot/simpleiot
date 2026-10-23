<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import { getText, getTextArray } from "#lib/siot/point.ts"
	import type { EditorProps } from "../types.ts"
	import RadioReceiverIcon from "@lucide/svelte/icons/radio-receiver"

	let props: EditorProps = $props()

	// point types the network manager client keeps that schema.ts does not list
	const typeInterface = "interface"
	const typeState = "state"
	const typeIPv4Gateway = "ipv4Gateway"
	const typeIPv4Addresses = "ipv4Addresses"
	const typeIPv4Netmasks = "ipv4Netmasks"
	const typeIPv4Nameservers = "ipv4Nameservers"

	const points = $derived(props.node.points)
	const stateDisplay = $derived(getText(points, typeState).replaceAll("NmDeviceState", ""))

	const rows = $derived([
		["State", stateDisplay],
		["IPv4 Gateway", getText(points, typeIPv4Gateway)],
		...getTextArray(points, typeIPv4Addresses).map((a) => ["IPv4 Addr", a]),
		...getTextArray(points, typeIPv4Netmasks).map((a) => ["IPv4 Netmask", a]),
		...getTextArray(points, typeIPv4Nameservers).map((a) => ["IPv4 NameServers", a]),
	])
</script>

<NodeEditor {props} icon={RadioReceiverIcon} description={getText(points, typeInterface)}>
	{#each rows as [label, value], i (i)}
		<div class="row">{label}: {value}</div>
	{/each}
</NodeEditor>

<style>
	.row {
		padding-left: 50px;
	}
</style>
