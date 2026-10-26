// The guard against losing unsaved point edits.

import { store } from "./siot/store.ts"

// confirmDiscard asks before pending edits are thrown away, and is true
// when there are none or the user agrees.
export function confirmDiscard(): boolean {
	if (!store.hasEdits) {
		return true
	}
	return window.confirm("You have unsaved changes. Discard them?")
}
