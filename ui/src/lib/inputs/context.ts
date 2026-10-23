// The inputs of a node editor read the node and report edits through a
// context the editor sets, so each input names only its point and label.

import { getContext, setContext } from "svelte"
import type { Node } from "#lib/siot/node.ts"
import type { Point } from "#lib/siot/point.ts"

export interface InputContext {
	readonly node: Node
	// now is the time in milliseconds that edits are stamped with
	readonly now: number
	// labelWidth is the width in pixels of the label column
	readonly labelWidth: number
	readonly scratch: string
	onEditPoints(points: Point[]): void
	onEditScratch(text: string): void
}

const key = Symbol("inputs")

export function setInputContext(ctx: InputContext) {
	setContext(key, ctx)
}

export function inputContext(): InputContext {
	const ctx = getContext<InputContext | undefined>(key)
	if (!ctx) {
		throw new Error("an input is used outside a node editor")
	}
	return ctx
}

// time is the ISO time to stamp an edit made now.
export const editTime = (ctx: InputContext) => new Date(ctx.now).toISOString()
