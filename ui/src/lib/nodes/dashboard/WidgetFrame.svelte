<script lang="ts">
	// WidgetFrame is the card around a widget: the component when the
	// registry has it, a placeholder naming it when not, and a mark when a
	// binding is dead so a dashboard synced to a device missing the node
	// still shows its shape.
	import type { Placed } from "./bindings.ts"
	import NestedDashboard from "./NestedDashboard.svelte"

	let { p, bare = false }: { p: Placed; bare?: boolean } = $props()

	const dead = $derived(Object.values(p.values).some((v) => v.state === "dead"))
	const C = $derived(p.descriptor?.component)
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
