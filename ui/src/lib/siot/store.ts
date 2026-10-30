// The page's one tree store, kept across route changes.

import { browserEnv } from "./browser-env.ts"
import { TreeStore } from "./tree.svelte.ts"

export const store = new TreeStore(browserEnv)
