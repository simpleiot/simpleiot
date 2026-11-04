// Temperature → liquid color: RGB lerp from cool blue to hot red (20–90 °C),
// deliberately avoiding the green midpoint an HSL hue sweep would produce.
const COOL = [0x39, 0x87, 0xe5]
const HOT = [0xe3, 0x49, 0x48]

export function tempColor(tC: number, darken = 0): string {
	const f = Math.max(0, Math.min(1, (tC - 20) / 70))
	const ch = COOL.map((c, i) => Math.round((c + ((HOT[i] ?? c) - c) * f) * (1 - darken)))
	return `rgb(${ch[0] ?? 0} ${ch[1] ?? 0} ${ch[2] ?? 0})`
}
