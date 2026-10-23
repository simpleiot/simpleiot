<script lang="ts">
	// A device credential sits under a device node on the upstream and holds
	// the device's public key. The upstream keeps lastConnect and connected
	// on it.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import type { Node } from "#lib/siot/node.ts"
	import { getBool, getText, getValue } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { toDateTimeString } from "#lib/util/iso8601.ts"
	import type { EditorProps } from "../types.ts"
	import KeyIcon from "@lucide/svelte/icons/key"

	let props: EditorProps = $props()

	const disabled = $derived(getBool(props.node.points, PointType.Disabled))
	const connected = $derived(getBool(props.node.points, PointType.Connected))
	const pending = $derived(getBool(props.node.points, PointType.Pending))
	const enrolledBy = $derived(enrolledByText())
	const lastConnect = $derived(getValue(props.node.points, PointType.LastConnect))

	// enrolledByText names the enrollment token a credential came through,
	// as its description and the node it sits in, so a credential from
	// another group's token stands out at approval time. A token the user
	// cannot see shows as its ID.
	function enrolledByText(): string | undefined {
		const id = getText(props.node.points, PointType.EnrolledBy)
		if (id === "") {
			return undefined
		}
		const token = props.findNode(id)
		if (!token) {
			return id
		}
		return nodeName(token) + " in " + placeName(token.parent)
	}

	function placeName(id: string): string {
		const n = props.findNode(id)
		return n ? nodeName(n) : id
	}

	function nodeName(n: Node): string {
		const desc = getText(n.points, PointType.Description)
		return desc === "" ? n.id : desc
	}

	function formatLastConnect(t: number): string {
		return t <= 0 ? "never" : toDateTimeString(Math.round(t * 1000))
	}
</script>

<NodeEditor {props} icon={KeyIcon} labelWidth={180}>
	{#snippet header()}
		{#if pending}
			<span class="pending"
				>{enrolledBy !== undefined
					? "(pending approval, enrolled with " + enrolledBy + ")"
					: "(pending approval)"}</span
			>
		{/if}
		{#if connected && !disabled && !pending}<span>(connected)</span>{/if}
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<TextInput
		type={PointType.PubKey}
		label="Public key"
		placeholder="from siot key show on the device"
	/>
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	{#if pending}
		<CheckboxInput type={PointType.Pending} label="Pending" />
		<div>Uncheck Pending to approve this device.</div>
	{/if}
	<div>Enrolled with: {enrolledBy ?? "added by hand"}</div>
	<div>Last connect: {formatLastConnect(lastConnect)}</div>
</NodeEditor>

<style>
	.pending {
		color: var(--orange);
	}
</style>
