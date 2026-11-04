// The reduced-motion preference, read from the system setting. nautilus
// keeps a three-way switch in localStorage; here the system setting is
// enough, and the CSS animations follow the same media query.
export function reducedMotion(): boolean {
	return typeof matchMedia === "function" && matchMedia("(prefers-reduced-motion: reduce)").matches
}
