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
		onWrite,
	}: {
		p: Placed
		bare?: boolean
		// onWrite sends a point to a node, for the widget's controls
		onWrite?: (nodeId: string, point: Point) => void
	} = $props()

	const dead = $derived(Object.values(p.values).some((v) => v.state === "dead"))
	const C = $derived(p.descriptor?.component)

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

<div class="frame" class:bare class:dead title={dead ? "a bound node was not found" : undefined}>
	{#if p.nested}
		<NestedDashboard id={p.nested} ancestors={p.ancestors} />
	{:else if C}
		<C
			label={p.label}
			values={p.values}
			options={p.options}
			width={bare ? p.layout.w : undefined}
			onWrite={write}
		/>
	{:else}
		<div class="unknown">
			<span class="name">{p.component}?</span>
			<span class="label">{p.label}</span>
		</div>
	{/if}
</div>

<style>
	.frame {
		box-sizing: border-box;
		height: 100%;
		padding: 12px 14px;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--detail-bg);
		overflow: hidden;
	}

	.dead {
		border-style: dashed;
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
