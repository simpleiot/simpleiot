// dashHref is the address of a dashboard page. via is the parent its
// relative bindings start from, carried as ?via= so a mirrored dashboard
// opens in the place it was reached through; "" leaves it to the page.
import { resolve } from "$app/paths"

export function dashHref(id: string, via = "", widget = ""): string {
	const path = resolve("/dash/[id]/[[widget]]", { id, widget: widget || undefined })
	return via === "" ? path : `${path}?via=${encodeURIComponent(via)}`
}
