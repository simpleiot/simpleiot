<script lang="ts">
	// Confirm is the row a control shows before a write when its confirm
	// option is on: what is about to be set, and the two buttons.
	let { text, onConfirm, onCancel }: { text: string; onConfirm: () => void; onCancel: () => void } =
		$props()
</script>

<div class="confirm" role="group" aria-label="Confirm {text}">
	<span>{text}</span>
	<button
		class="btn blue"
		type="button"
		onclick={(e) => {
			e.stopPropagation()
			onConfirm()
		}}>confirm</button
	>
	<button
		class="btn gray"
		type="button"
		onclick={(e) => {
			e.stopPropagation()
			onCancel()
		}}>cancel</button
	>
</div>

<style>
	.confirm {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
		font-size: 14px;
	}
</style>
