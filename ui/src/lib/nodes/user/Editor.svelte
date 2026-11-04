<script lang="ts">
	import NodeEditor from "#lib/components/NodeEditor.svelte"
	import KeyValueInput from "#lib/inputs/KeyValueInput.svelte"
	import NodeRefInput from "#lib/inputs/NodeRefInput.svelte"
	import TextInput from "#lib/inputs/TextInput.svelte"
	import { blankMagicValue, getText, newText, type Point } from "#lib/siot/point.ts"
	import type { NodeMatch } from "#lib/siot/search.ts"
	import { NodeType, PointType } from "#lib/siot/schema.ts"
	import type { EditorProps } from "../types.ts"
	import Scope from "./Scope.svelte"
	import UserIcon from "@lucide/svelte/icons/user"

	let props: EditorProps = $props()

	const name = $derived(
		getText(props.node.points, PointType.FirstName) +
			" " +
			getText(props.node.points, PointType.LastName)
	)

	// A dashboard can sit in more than one place. Picking one records the
	// place it was found in, which is where its relative bindings start.
	const pickHome = (m: NodeMatch) =>
		props.onEditPoints([
			newText(PointType.HomeVia, "0", m.parent, new Date(props.now).toISOString()),
		])

	const lowerCase = (points: Point[]) => points.map((p) => ({ ...p, text: p.text.toLowerCase() }))

	// passwords are stored as bcrypt hashes; show the field blank instead of
	// the stored hash, so typing always sets a new password
	const blankHashedPass = (points: Point[]) =>
		points.map((p) =>
			p.type === PointType.Pass && p.text.startsWith("$2") ? { ...p, text: blankMagicValue } : p
		)
</script>

<NodeEditor {props} icon={UserIcon} labelWidth={100} description={name}>
	<TextInput type={PointType.FirstName} label="First Name" />
	<TextInput type={PointType.LastName} label="Last Name" />
	<Scope edit={lowerCase}>
		<TextInput type={PointType.Email} label="Email/User" />
	</Scope>
	<TextInput type={PointType.Phone} label="Phone" />
	<Scope view={blankHashedPass}>
		<TextInput type={PointType.Pass} label="Pass" />
	</Scope>
	<NodeRefInput
		type={PointType.HomeDashboard}
		label="Home dashboard"
		search={(query: string) => props.searchNodes(query, NodeType.Dashboard)}
		resolve={props.resolveNode}
		findNode={props.findNode}
		onPick={pickHome}
	/>
	<NodeRefInput
		type={PointType.HomeVia}
		label="Home dashboard under"
		search={(query: string) => props.searchNodes(query)}
		resolve={props.resolveNode}
		findNode={props.findNode}
	/>
	<KeyValueInput type={PointType.Tag} label="Tags" buttonLabel="Add Tag" />
</NodeEditor>
