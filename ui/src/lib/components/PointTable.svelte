<script lang="ts">
	// PointTable lists formatted points in two columns.
	import type { Row } from "./pointFormat.ts"

	let { rows, label }: { rows: Row[]; label: string } = $props()
</script>

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
