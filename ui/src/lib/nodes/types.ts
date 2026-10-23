// What a node editor is given. Editors take props and report edits through
// callbacks; only the page talks to the server.

import type { Node } from "#lib/siot/node.ts"
import type { Point } from "#lib/siot/point.ts"
import type { TreeNode } from "#lib/siot/tree.ts"

// Clipboard is a node copied for pasting elsewhere.
export interface Clipboard {
	id: string
	// parent is the node it was copied from
	parent: string
	description: string
}

// GeneratedToken is a token just made for an enrollment token node. It is
// shown once, on that node, until the page is left.
export interface GeneratedToken {
	id: string
	token: string
}

export interface EditorProps {
	node: Node
	parent: Node | undefined
	children: TreeNode[]
	// now is the time in milliseconds, updated once a second
	now: number
	// zone is the browser's time zone name
	zone: string
	expDetail: boolean
	// modified is set while the node has unsaved edits
	modified: boolean
	copy: Clipboard | null
	scratch: string
	generatedToken: GeneratedToken | null
	// findNode looks up a node on the page by ID
	findNode: (id: string) => Node | undefined
	// onEditPoints records edits to save; onSendPoints sends points at once
	onEditPoints: (points: Point[]) => void
	onSendPoints: (points: Point[]) => void
	onEditScratch: (text: string) => void
	onUploadFile: (binary: boolean) => void
	onGenerateKey: () => void
}
