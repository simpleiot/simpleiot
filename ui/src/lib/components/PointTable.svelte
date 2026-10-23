<script lang="ts">
	// PointTable lists formatted points in two columns.
	import type { Row } from "./pointFormat.ts"

	let { rows, label }: { rows: Row[]; label: string } = $props()
</script>

<table>
	<thead>
		<tr><th>{label}</th><th>Value</th></tr>
	</thead>
	<tbody>
		{#each rows as row, i (i)}
			<tr><td>{row.desc}</td><td class="value">{row.value}</td></tr>
		{/each}
	</tbody>
</table>

<style>
	table {
		border-collapse: collapse;
		margin: 7px;
		align-self: flex-start;
	}

	th,
	td {
		border: 1px solid var(--fg);
		padding: 5px 15px;
		overflow-wrap: anywhere;
	}

	th {
		font-weight: 700;
		text-align: center;
	}

	.value {
		text-align: right;
	}
</style>
