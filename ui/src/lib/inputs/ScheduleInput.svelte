<script lang="ts">
	// ScheduleInput edits a schedule: start and end times, and either
	// weekdays or a list of dates, shown in the browser's zone and stored
	// in UTC.
	import X from "@lucide/svelte/icons/x"
	import IconButton from "#lib/components/IconButton.svelte"
	import { DataType } from "#lib/siot/point.ts"
	import { PointType } from "#lib/siot/schema.ts"
	import { date as sanitizeDate, time as sanitizeTime } from "#lib/util/sanitize.ts"
	import type { Schedule } from "#lib/util/time.ts"
	import Button from "./Button.svelte"
	import { editTime, inputContext } from "./context.ts"
	import Label from "./Label.svelte"
	import { pointsToSchedule, scheduleToPoints, toLocal, toUTC, weekdays } from "./schedule.ts"

	const ctx = inputContext()
	const startId = $props.id()

	// minutes east of UTC at the edit time
	const offset = $derived(-new Date(ctx.now).getTimezoneOffset())
	const local = $derived(toLocal(offset, pointsToSchedule(ctx.node.points)))
	const anyWeekday = $derived(local.weekdays.length > 0)

	// weekday numbers match Go's time.Weekday
	const dayLetters = ["S", "M", "T", "W", "T", "F", "S"]
	const dayNames = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]

	function send(s: Schedule) {
		ctx.onEditPoints(scheduleToPoints(editTime(ctx), toUTC(offset, s)))
	}

	function setWeekday(d: number, on: boolean) {
		const ws = on ? [...new Set([...local.weekdays, d])] : local.weekdays.filter((w) => w !== d)
		send({ ...local, weekdays: ws.sort((a, b) => a - b) })
	}

	function addDate() {
		ctx.onEditPoints([
			{
				type: PointType.Date,
				key: String(local.dateCount),
				time: editTime(ctx),
				dataType: DataType.String,
				value: 0,
				text: "",
				tombstone: 0,
			},
		])
	}
</script>

<div class="schedule">
	{#if local.dates.length === 0}
		<fieldset class="weekdays" style:padding-left="{Math.max(ctx.labelWidth - 120, 0)}px">
			<legend>Weekdays:</legend>
			{#each weekdays as d (d)}
				<label>
					<span aria-hidden="true">{dayLetters[d]}</span>
					<input
						type="checkbox"
						aria-label={dayNames[d]}
						checked={local.weekdays.includes(d)}
						onchange={(e) => {
							setWeekday(d, e.currentTarget.checked)
						}}
					/>
				</label>
			{/each}
		</fieldset>
	{/if}
	<Label label="Start time" id="{startId}-start">
		<input
			id="{startId}-start"
			type="text"
			value={local.startTime}
			oninput={(e) => {
				send({ ...local, startTime: sanitizeTime(e.currentTarget.value) })
			}}
		/>
	</Label>
	<Label label="End time" id="{startId}-end">
		<input
			id="{startId}-end"
			type="text"
			value={local.endTime}
			oninput={(e) => {
				send({ ...local, endTime: sanitizeTime(e.currentTarget.value) })
			}}
		/>
	</Label>
	{#if !anyWeekday}
		<div class="dates" style:padding-left="{Math.max(ctx.labelWidth - 59, 0)}px">
			<span>Dates:</span>
			{#each local.dates as d, i (i)}
				<div class="date">
					<input
						type="text"
						aria-label="Date {i + 1}"
						placeholder="YYYY-MM-DD"
						value={d}
						oninput={(e) => {
							const v = sanitizeDate(e.currentTarget.value)
							send({ ...local, dates: local.dates.map((x, j) => (j === i ? v : x)) })
						}}
					/>
					<IconButton
						label="Remove date {i + 1}"
						onclick={() => {
							send({ ...local, dates: local.dates.filter((_, j) => j !== i) })
						}}><X size={20} /></IconButton
					>
				</div>
			{/each}
			<div><Button onclick={addDate}>Add Date</Button></div>
		</div>
	{/if}
</div>

<style>
	.schedule {
		display: flex;
		flex-direction: column;
		gap: 5px;
	}

	.weekdays {
		border: none;
		margin: 0;
		padding: 0 0 5px;
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 20px;
	}

	.weekdays legend {
		float: left;
		padding: 0;
	}

	.weekdays label {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.dates {
		display: flex;
		flex-direction: column;
		gap: 5px;
	}

	.date {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	@media (max-width: 600px) {
		.dates,
		.weekdays {
			padding-left: 0 !important;
		}
	}
</style>
