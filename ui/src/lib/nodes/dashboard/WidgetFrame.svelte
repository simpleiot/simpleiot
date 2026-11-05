<script lang="ts">
	// WidgetFrame is the card around a widget: the component when the
	// registry has it, a placeholder naming it when not, and a mark when a
	// binding is dead so a dashboard synced to a device missing the node
	// still shows its shape. A widget whose bound node accepts writes gets
	// an onWrite, and acts as a control.
	import { newText, newValue, type Point } from "#lib/siot/point.ts"
	import type { Write } from "#lib/widgets/registry.ts"
	import type { Placed } from "./bindings.ts"
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
					onWrite(
						target.nodeId,
						w.text !== ""
							? newText(target.type, target.key, w.text, t)
							: newValue(target.type, target.key, w.value, t)
					)
				}
			: undefined
	)
</script>

<div
	class="frame"
	class:bare
	class:dead
	class:editing
	title={dead ? "a bound node was not found" : undefined}
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
