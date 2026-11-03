<script lang="ts">
	// A status pill: on or off from a value, never color alone.
	import type { WidgetProps } from "../registry.ts"

	let { label, values, options }: WidgetProps = $props()

	const v = $derived(values.value)
	const dead = $derived(!v || v.state === "dead")
	const on = $derived(!dead && (v?.value ?? 0) !== 0)
</script>

<div class="status">
	<span class="pill" class:on class:dead>
		<span class="icon" aria-hidden="true">{dead ? "○" : on ? "●" : "○"}</span>
		{dead ? "no data" : on ? options.on : options.off}
	</span>
	<span class="label">{label}</span>
</div>

<style>
	.status {
		display: flex;
		flex-direction: column;
		gap: 6px;
		height: 100%;
	}

	.pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		align-self: flex-start;
		padding: 4px 12px;
		border: 1px solid var(--border);
		border-radius: 999px;
		font-weight: 550;
		background: var(--detail-bg);
		--c: var(--muted);
	}

	.pill.on {
		--c: var(--green);
	}

	.icon {
		color: var(--c);
	}

	.dead {
		color: var(--muted);
	}

	.label {
		font-size: 13px;
		color: var(--muted);
	}
</style>
