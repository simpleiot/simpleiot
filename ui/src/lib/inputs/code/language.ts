// languageOf names the language a file is written in from its name, for
// the code editor's highlighting; undefined when the name says nothing.
export type Language = "xml" | "yaml" | "json" | "javascript" | "css"

const byExtension: Readonly<Record<string, Language>> = {
	svg: "xml",
	xml: "xml",
	html: "xml",
	yaml: "yaml",
	yml: "yaml",
	json: "json",
	js: "javascript",
	mjs: "javascript",
	ts: "javascript",
	css: "css",
}

export function languageOf(name: string): Language | undefined {
	const dot = name.lastIndexOf(".")
	if (dot < 0) {
		return undefined
	}
	return byExtension[name.slice(dot + 1).toLowerCase()]
}
