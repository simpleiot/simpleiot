<script lang="ts">
	// The frame draws the nested dashboard itself; this is what shows when
	// the widget has no dashboard bound yet.
	import type { WidgetProps } from "../registry.ts"

	let { label }: WidgetProps = $props()
</script>

<div class="placeholder">
	<span>nested dashboard</span>
	{#if label}<span class="label">{label}</span>{/if}
	<span class="hint">bind a dashboard node</span>
</div>

<style>
	.placeholder {
		display: flex;
		flex-direction: column;
		gap: 4px;
		color: var(--muted);
	}

	.label {
		font-weight: 600;
	}

	.hint {
		font-size: 13px;
	}
</style>
