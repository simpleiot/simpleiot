<script lang="ts">
	// WidgetFrame is the card around a widget: the component when the
	// registry has it, a placeholder naming it when not, and a mark when a
	// binding is dead so a dashboard synced to a device missing the node
	// still shows its shape. A widget whose bound node accepts writes gets
	// an onWrite, and acts as a control. A widget with a navigate binding
	// opens that dashboard when clicked, where the widget does not act on
	// the click itself; one with a url option opens that address in a new
	// tab instead.
	import { goto } from "$app/navigation"
	import type { Point } from "#lib/siot/point.ts"
	import { background, navigateBinding, type Write } from "#lib/widgets/registry.ts"
	import { writePoint, type Placed } from "./bindings.ts"
	import { linkUrl } from "./link.ts"
	import NestedDashboard from "./NestedDashboard.svelte"

	let {
		p,
		bare = false,
		editing = false,
		onWrite,
	}: {
		p: Placed
		bare?: boolean
		// editing is the dashboard's edit mode
		editing?: boolean
		// onWrite sends a point to a node, for the widget's controls
		onWrite?: (nodeId: string, point: Point) => void
	} = $props()

	const dead = $derived(Object.values(p.values).some((v) => v.state === "dead"))
	const C = $derived(p.descriptor?.component)

	// scale enlarges or shrinks the widget's content, for a tile that
	// should read from across the room or sit small in a corner. On a
	// canvas a widget that is not a drawing also follows its width: the
	// corner handle then sizes its text as it does a tank's shell.
	const option = $derived.by(() => {
		const s = Number(p.options.scale)
		return Number.isFinite(s) && s > 0 ? s : 1
	})
	const byWidth = $derived(p.descriptor?.fit === "width")
	// fill is the card drawn behind the widget, so a value placed over a
	// tank or a drawing reads cleanly
	const fill = $derived(background(p.options))
	const scale = $derived(
		bare && !byWidth ? (p.layout.w / (p.descriptor?.width ?? 160)) * option : option
	)

	// a write from the widget goes to the target its binding names
	const write = $derived(
		onWrite && Object.keys(p.writes).length > 0
			? (prop: string, w: Write) => {
					const target = p.writes[prop]
					if (!target) {
						return
					}
					const t = new Date().toISOString()
					onWrite(target.nodeId, writePoint(target, p.values[prop], w, t, p.options))
				}
			: undefined
	)

	// the dashboard a click opens, else the address the url option names;
	// none in edit mode, where a click selects
	const dash = $derived(editing ? undefined : p.opens[navigateBinding])
	const url = $derived(
		editing || dash !== undefined ? undefined : linkUrl(p.options.url ?? "", location)
	)
	const href = $derived(dash ?? url)

	function go() {
		if (dash !== undefined) {
			// the path is resolved in dashHref; only the query is appended
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			void goto(dash)
		} else if (url !== undefined) {
			window.open(url, "_blank", "noopener")
		}
	}

	// a widget that acts on a click stops it, and a link inside the widget,
	// such as a nested dashboard's title, goes where it points
	function open(e: MouseEvent) {
		if (href === undefined || (e.target as Element | null)?.closest("a")) {
			return
		}
		go()
	}

	function onkeydown(e: KeyboardEvent) {
		if (href !== undefined && e.key === "Enter" && e.target === e.currentTarget) {
			go()
		}
	}
</script>

<!-- the frame takes focus only when it is a link, which the check cannot
     tell from the conditional role -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
	class="frame"
	class:bare
	class:dead
	class:editing
	class:filled={fill !== undefined}
	style:background={fill}
	class:opens={href !== undefined}
	title={dead ? "a bound node was not found" : url}
	role={href !== undefined ? "link" : undefined}
	tabindex={href !== undefined ? 0 : undefined}
	onclick={open}
	{onkeydown}
>
	{#if p.nested}
		<NestedDashboard id={p.nested} ancestors={p.ancestors} via={p.via} />
	{:else if C}
		<div class="content" class:sized={!bare} style:zoom={scale}>
			<C
				label={p.label}
				values={p.values}
				options={p.options}
				width={bare && byWidth ? p.layout.w / scale : undefined}
				onWrite={write}
				writable={Object.keys(p.writes)}
				opens={editing ? undefined : p.opens}
				control={Object.keys(p.writes).length > 0}
				{editing}
			/>
		</div>
	{:else}
		<div class="unknown">
			<span class="name">{p.component}?</span>
			<span class="label">{p.label}</span>
		</div>
	{/if}
</div>

<style>
	/* in view mode only the widget shows, as on a canvas; the card around
	   it appears in edit mode, where it is what is dragged, and a dashed
	   one marks a widget whose node was not found */
	.frame {
		box-sizing: border-box;
		height: 100%;
		padding: 12px 14px;
		border: 1px solid transparent;
		border-radius: 8px;
		background: none;
		overflow: hidden;
	}

	.frame.editing {
		border-color: var(--border);
		background: var(--detail-bg);
	}

	.opens {
		cursor: pointer;
	}

	.opens:focus-visible {
		outline: 2px solid var(--blue);
	}

	.dead {
		border-color: var(--muted);
		border-style: dashed;
	}

	.content {
		height: 100%;
	}

	/* on a grid the tile's height sets the size of a value or a pill, so
	   a taller tile reads from further away; a canvas widget keeps the
	   fixed sizes, since its box has no height of its own */
	.content.sized {
		container-type: size;
		--num-size: clamp(22px, 34cqh, 200px);
		--pill-size: clamp(15px, 18cqh, 100px);
	}

	.bare {
		padding: 0;
		border: none;
		background: none;
		overflow: visible;
	}

	/* a card behind a canvas widget leaves room around its content */
	.bare.filled {
		padding: 4px 8px;
		border-radius: 6px;
	}

	.bare.dead {
		outline: 1px dashed var(--muted);
	}

	.unknown {
		display: flex;
		flex-direction: column;
		gap: 4px;
		color: var(--muted);
	}

	.name {
		font-family: var(--mono);
	}

	.label {
		font-size: 13px;
	}
</style>
