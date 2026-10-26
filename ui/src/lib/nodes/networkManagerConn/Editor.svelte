<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import NumberInput from "#lib/inputs/NumberInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import CableIcon from "@lucide/svelte/icons/cable"

	let props: EditorProps = $props()

	// point types the network manager client keeps that schema.ts does not list
	const typeInterface = "interface"
	const typeAutoConnect = "autoConnect"
	const typeAutoConnectPriority = "autoConnectPriority"
	const typeIPv4Config = "ipv4Config"
	const typeIPv6Config = "ipv6Config"
</script>

<NodeEditor {props} icon={CableIcon}>
	<TextInput type={PointType.Description} label="Description" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<TextInput type={typeInterface} label="Interface" />
	<CheckboxInput type={typeAutoConnect} label="Auto connect" />
	<NumberInput type={typeAutoConnectPriority} label="Auto conn priority" />
	<div class="section">IPv4 Settings</div>
	<CheckboxInput type={typeIPv4Config} key="staticIP" label="Static IP" />
	<TextInput type={typeIPv4Config} key="address" label="Address" placeholder="1.2.3.4" />
	<TextInput type={typeIPv4Config} key="netmask" label="Netmask" placeholder="255.255.0.0" />
	<TextInput type={typeIPv4Config} key="gateway" label="Gateway" placeholder="10.0.0.1" />
	<TextInput type={typeIPv4Config} key="dnsServer1" label="DNS Server 1" placeholder="8.8.8.8" />
	<TextInput type={typeIPv4Config} key="dnsServer2" label="DNS Server 2" placeholder="8.8.4.4" />
	<div class="section">IPv6 Settings</div>
	<CheckboxInput type={typeIPv6Config} key="staticIP" label="Static IP" />
	<TextInput type={typeIPv6Config} key="address" label="Address" />
	<NumberInput type={typeIPv6Config} key="prefix" label="Prefix" />
	<TextInput type={typeIPv6Config} key="gateway" label="Gateway" />
	<TextInput type={typeIPv6Config} key="dnsServer1" label="DNS Server 1" />
	<TextInput type={typeIPv6Config} key="dnsServer2" label="DNS Server 2" />
</NodeEditor>

<style>
	.section {
		font-weight: bold;
		text-align: center;
	}
</style>
