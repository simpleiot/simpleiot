<script lang="ts">
	// NodeRefInput picks a node for a point that holds a node ID. Typing
	// searches on the server, so a node need not be loaded in the tree; the
	// chosen node's ID is stored, and the field shows its description with
	// a link to it. The node on the clipboard, when there is one, is offered
	// first.
	import { resolve as resolvePath } from "$app/paths"
	import XIcon from "@lucide/svelte/icons/x"
	import IconButton from "#lib/components/IconButton.svelte"
	import type { Node } from "#lib/siot/node.ts"
	import { getBestDesc, getText, newText } from "#lib/siot/point.ts"
	import { moveActive, type NodeMatch } from "#lib/siot/search.ts"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"

	let {
		type,
		key = "0",
		label,
		search,
		resolve,
		findNode,
		suggest = null,
	}: {
		type: string
		key?: string
		label: string
		search: (query: string) => Promise<NodeMatch[]>
		resolve: (id: string) => Promise<Node | undefined>
		// findNode looks among the nodes on the page, which stay live
		findNode: (id: string) => Node | undefined
		suggest?: { id: string; description: string } | null
	} = $props()

	const ctx = inputContext()
	const id = $props.id()
	const listId = `${id}-list`

	const value = $derived(getText(ctx.node.points, type, key))

	// ---- the chosen node

	const loaded = $derived(value === "" ? undefined : findNode(value))
	let fetched = $state<{ id: string; node: Node | undefined } | null>(null)
	$effect(() => {
		const v = value
		if (v === "" || loaded || fetched?.id === v) {
			return
		}
		void resolve(v).then((node) => {
			fetched = { id: v, node }
		})
	})
	const target = $derived(loaded ?? (fetched?.id === value ? fetched.node : undefined))
	const pending = $derived(value !== "" && !loaded && fetched?.id !== value)

	// ---- searching

	let query = $state("")
	let open = $state(false)
	let searching = $state(false)
	let matches = $state<NodeMatch[]>([])
	let active = $state(-1)
	let timer: ReturnType<typeof setTimeout> | null = null
	let seq = 0

	const list = $derived.by((): NodeMatch[] => {
		if (query.trim() === "") {
			return suggest
				? [
						{
							id: suggest.id,
							type: "",
							description: suggest.description,
							path: ["clipboard"],
							rank: 0,
						},
					]
				: []
		}
		return matches
	})

	function set(nodeId: string) {
		ctx.onEditPoints([newText(type, key, nodeId, editTime(ctx))])
		query = ""
		matches = []
		open = false
		active = -1
	}

	function oninput(e: Event & { currentTarget: HTMLInputElement }) {
		query = e.currentTarget.value
		open = true
		active = -1
		if (timer) {
			clearTimeout(timer)
		}
		const q = query.trim()
		const mine = ++seq
		if (q === "") {
			matches = []
			return
		}
		timer = setTimeout(() => {
			searching = true
			search(q)
				.then((m) => {
					if (mine === seq) {
						matches = m
						active = m.length > 0 ? 0 : -1
					}
				})
				.catch(() => {
					if (mine === seq) {
						matches = []
					}
				})
				.finally(() => {
					if (mine === seq) {
						searching = false
					}
				})
		}, 150)
	}

	function onkeydown(e: KeyboardEvent) {
		switch (e.key) {
			case "ArrowDown":
			case "ArrowUp":
				open = true
				active = moveActive(active, e.key, list.length)
				e.preventDefault()
				break
			case "Enter": {
				const m = list[active]
				if (open && m) {
					set(m.id)
					e.preventDefault()
				}
				break
			}
			case "Escape":
				if (open) {
					open = false
					e.stopPropagation()
				}
				break
		}
	}
</script>

<Label {label} {id} edited={ctx.edited(type, key)}>
	{#if value === ""}
		<div class="picker">
			<input
				{id}
				type="text"
				role="combobox"
				aria-expanded={open && list.length > 0}
				aria-controls={listId}
				aria-autocomplete="list"
				aria-activedescendant={active >= 0 ? `${listId}-${active}` : undefined}
				autocomplete="off"
				placeholder="type a description to search"
				value={query}
				{oninput}
				{onkeydown}
				onfocus={() => (open = true)}
				onblur={() => (open = false)}
			/>
			{#if open && (list.length > 0 || searching)}
				<ul class="matches" id={listId} role="listbox">
					{#each list as m, i (m.id)}
						<li
							id="{listId}-{i}"
							role="option"
							aria-selected={i === active}
							class:active={i === active}
							onmousedown={(e) => {
								e.preventDefault()
								set(m.id)
							}}
						>
							<span class="desc">{m.description}</span>
							{#if m.type !== ""}<span class="type">{m.type}</span>{/if}
							{#if m.path.length > 0}<span class="path">{m.path.join(" / ")}</span>{/if}
						</li>
					{/each}
					{#if searching}
						<li class="status">searching...</li>
					{/if}
				</ul>
			{/if}
		</div>
	{:else}
		<div class="chip" class:not-found={!target && !pending}>
			{#if target}
				<a href={resolvePath("/nodes/[id]", { id: value })}>{getBestDesc(target.points)}</a>
			{:else if pending}
				<span>loading...</span>
			{:else}
				<span>not found</span>
			{/if}
			<code class="id" title={value}>{value.slice(0, 8)}</code>
			<IconButton label="Clear" onclick={() => set("")}><XIcon size={16} /></IconButton>
		</div>
	{/if}
</Label>

<style>
	.picker {
		position: relative;
		flex: 1;
		min-width: 200px;
		max-width: 480px;
	}

	.picker input {
		width: 100%;
		box-sizing: border-box;
	}

	.matches {
		position: absolute;
		z-index: 10;
		left: 0;
		right: 0;
		margin: 2px 0 0;
		padding: 4px 0;
		list-style: none;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: 4px;
		box-shadow: 0 4px 12px rgb(0 0 0 / 0.15);
		max-height: 280px;
		overflow-y: auto;
	}

	.matches li {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 2px 10px;
		padding: 6px 10px;
		cursor: pointer;
	}

	.matches li.active,
	.matches li:hover {
		background: var(--hover);
	}

	.matches .status {
		color: var(--muted);
		cursor: default;
	}

	.type,
	.path {
		font-size: 0.85em;
		color: var(--muted);
	}

	.path {
		flex-basis: 100%;
	}

	.chip {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 2px 4px 2px 10px;
		border-radius: 16px;
		background: var(--badge-bg);
		color: var(--badge-fg);
	}

	.chip a {
		color: inherit;
	}

	.chip.not-found {
		background: var(--orange);
		color: #000000;
	}

	.id {
		font-family: var(--mono);
		font-size: 0.85em;
		opacity: 0.7;
	}
</style>
