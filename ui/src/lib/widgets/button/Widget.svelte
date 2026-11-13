<script lang="ts">
	// A button whose navigate binding names a dashboard; the frame around it
	// opens the dashboard when it is clicked. It reads the text option, else
	// the widget's label, which is the dashboard's description when the
	// widget has none of its own.
	import { navigateBinding, type WidgetProps } from "../registry.ts"

	let { label, values, options, editing = false }: WidgetProps = $props()

	const text = $derived((options.text ?? "").trim() || label || "open")
	const bound = $derived(values[navigateBinding])
	const dead = $derived(!bound || bound.state === "dead")
</script>

<div class="host">
	<span
		class="button"
		class:dead={!editing && dead}
		title={dead ? "bind a dashboard to navigate" : undefined}>{text}</span
	>
</div>

<style>
	.host {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
	}

	.button {
		display: inline-block;
		box-sizing: border-box;
		max-width: 100%;
		padding: 8px 16px;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--bg);
		color: var(--fg);
		font-weight: 600;
		text-align: center;
		overflow-wrap: anywhere;
	}

	/* the frame takes the click and draws the focus ring */
	:global(.opens:hover) .button {
		border-color: var(--blue);
		color: var(--blue);
	}

	.dead {
		color: var(--muted);
		border-style: dashed;
	}
</style>
