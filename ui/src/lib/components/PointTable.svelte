<script lang="ts">
	// PointTable lists formatted points in two columns. A collapsible table
	// starts closed behind a toggle, for editors that list many points.
	import ChevronDown from "@lucide/svelte/icons/chevron-down"
	import ChevronRight from "@lucide/svelte/icons/chevron-right"
	import type { Row } from "./pointFormat.ts"

	let {
		rows,
		label,
		collapsible = false,
	}: { rows: Row[]; label: string; collapsible?: boolean } = $props()

	let open = $state(false)
	const shown = $derived(!collapsible || open)
</script>

{#if collapsible}
	<button type="button" class="toggle" aria-expanded={open} onclick={() => (open = !open)}>
		{#if open}<ChevronDown size={16} />{:else}<ChevronRight size={16} />{/if}
		{label}s ({rows.length})
	</button>
{/if}
{#if shown}
	<table class="data-table">
		<thead>
			<tr><th>{label}</th><th class="num">Value</th></tr>
		</thead>
		<tbody>
			{#each rows as row, i (i)}
				<tr><td>{row.desc}</td><td class="num">{row.value}</td></tr>
			{/each}
		</tbody>
	</table>
{/if}

<style>
	.toggle {
		display: flex;
		align-items: center;
		align-self: flex-start;
		gap: 4px;
		padding: 2px 0;
		border: none;
		background: none;
		color: var(--muted);
		font: inherit;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
	}

	.toggle:hover {
		color: inherit;
	}
</style>
