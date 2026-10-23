<script lang="ts">
	// An enrollment token lets devices that hold it ask this instance for a
	// credential. Only a hash is stored; the token is shown once when
	// generated.
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import Value from "#lib/inputs/Value.svelte"
	import Button from "#lib/inputs/Button.svelte"
	import CheckboxInput from "#lib/inputs/CheckboxInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { getText, getValue } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { toDateTimeString } from "#lib/util/iso8601.ts"
	import type { EditorProps } from "../types.ts"
	import KeyIcon from "@lucide/svelte/icons/key"

	let props: EditorProps = $props()

	const hash = $derived(getText(props.node.points, PointType.TokenHash))
	const expires = $derived(getValue(props.node.points, PointType.Expires))
	const generated = $derived(
		props.generatedToken?.id === props.node.id ? props.generatedToken.token : undefined
	)

	function formatExpires(t: number): string {
		return t <= 0 ? "never" : toDateTimeString(Math.round(t * 1000))
	}
</script>

<NodeEditor {props} icon={KeyIcon} labelWidth={180}>
	{#snippet header()}
		<span>(enrollment token)</span>
	{/snippet}
	<TextInput type={PointType.Description} label="Description" />
	<CheckboxInput type={PointType.Disabled} label="Disabled" />
	<CheckboxInput type={PointType.AutoApprove} label="Auto approve" />
	<div>Auto approve lets enrolled devices in without an operator.</div>
	<Value label="Expires">{formatExpires(expires)}</Value>
	<div>{hash === "" ? "No token generated yet" : "Token set (only its hash is stored)"}</div>
	{#if hash === ""}
		<div class="button-row">
			<Button color="blue" onclick={props.onGenerateKey}>Generate token</Button>
		</div>
	{/if}
	{#if generated !== undefined}
		<div class="token">
			<div>Enrollment token. It is shown once and not stored here.</div>
			<div class="mono">{generated}</div>
			<div>Put it on each device's sync node as the Enroll Token.</div>
		</div>
	{/if}
</NodeEditor>

<style>
	.token {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 10px;
		border: 1px solid var(--orange);
	}

	.mono {
		font-family: var(--mono);
		font-size: 14px;
		overflow-wrap: anywhere;
	}
</style>
