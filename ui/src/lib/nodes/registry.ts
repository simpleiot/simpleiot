// The node type registry. Each node type is a directory under
// src/lib/nodes holding its editor and an index.ts that exports a
// descriptor made with defineNodeType; index.ts in this directory collects
// them. The tree page and store ask the registry and name no types.

import type { Component } from "svelte"
import type { EditorProps } from "./types.ts"

export type Icon = Component<{ size?: number | string }>

export interface NodeTypeDescriptor {
	type: string
	// label names the type in the add list and the raw view
	label: string
	icon: Icon
	// parents are the node types it may be added under
	parents: readonly string[]
	// hasChildren is set for types that take child nodes
	hasChildren?: boolean
	// ownedBy is the parent type the node must live under to work; it may
	// only be mirrored elsewhere
	ownedBy?: string
	// sortKey orders siblings of different types; the type name otherwise
	sortKey?: string
	editor?: Component<EditorProps>
}

export function defineNodeType(d: NodeTypeDescriptor): NodeTypeDescriptor {
	return d
}

export interface Registry {
	all: readonly NodeTypeDescriptor[]
	get(type: string): NodeTypeDescriptor | undefined
	editorFor(type: string): Component<EditorProps> | undefined
	// addableUnder lists the types that may be added under a parent type,
	// in sort order
	addableUnder(parentType: string): NodeTypeDescriptor[]
	hasChildren(type: string): boolean
	ownerOf(type: string): string | undefined
	sortKey(type: string): string
}

export function createRegistry(descriptors: readonly NodeTypeDescriptor[]): Registry {
	const byType = new Map(descriptors.map((d) => [d.type, d]))
	const sortKey = (type: string) => byType.get(type)?.sortKey ?? type
	const all = descriptors
		.slice()
		.sort((a, b) => cmp(sortKey(a.type), sortKey(b.type)) || cmp(a.label, b.label))
	return {
		all,
		get: (type) => byType.get(type),
		editorFor: (type) => byType.get(type)?.editor,
		addableUnder: (parentType) => all.filter((d) => d.parents.includes(parentType)),
		hasChildren: (type) => byType.get(type)?.hasChildren ?? false,
		ownerOf: (type) => byType.get(type)?.ownedBy,
		sortKey,
	}
}

const cmp = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0)
