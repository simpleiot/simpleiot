// sanitize cleans an uploaded SVG before it is injected into the page. A
// file node is writable by anyone with write access to its group and a
// dashboard is viewed by everyone in the group with their own sign-in, so
// an unsanitized drawing is a stored script injection from one user to
// another. DOMPurify's SVG profile removes scripts, event handlers, and
// unknown elements; the hooks below also remove anything that would reach
// out of the page: links, embedded documents, and references to anything
// but a fragment of the drawing itself or an inline data image.
import DOMPurify from "dompurify"

// local says whether a reference stays inside the page
function local(value: string): boolean {
	const v = value.trim()
	return v === "" || v.startsWith("#") || /^data:image\//i.test(v)
}

// externalUrl finds a url() that is not a fragment reference
const externalUrl = /url\(\s*(['"]?)(?!\s*#)[^)'"]/i

DOMPurify.addHook("uponSanitizeAttribute", (_node, data) => {
	const name = data.attrName.toLowerCase()
	if ((name === "href" || name === "xlink:href") && !local(data.attrValue)) {
		data.keepAttr = false
		return
	}
	if (externalUrl.test(data.attrValue)) {
		data.keepAttr = false
	}
})

DOMPurify.addHook("uponSanitizeElement", (node, data) => {
	if (data.tagName !== "style") {
		return
	}
	// a stylesheet may neither import nor reference anything outside the
	// drawing; a rule that does is rewritten rather than dropped so the
	// rest of the sheet still applies
	node.textContent = (node.textContent ?? "")
		.replace(/@import[^;]*;?/gi, "")
		.replace(/url\(\s*(['"]?)(?!\s*#)[^)]*\)/gi, "none")
})

const config = {
	USE_PROFILES: { svg: true, svgFilters: true },
	// use is left out of the profile because it can reference another
	// document; the href hook limits it to the drawing itself
	ADD_TAGS: ["style", "use"],
	FORBID_TAGS: ["a", "foreignObject", "script"],
}

// sanitizeSvg returns markup safe to inject, or "" when nothing is left.
export function sanitizeSvg(text: string): string {
	return DOMPurify.sanitize(text, config)
}
