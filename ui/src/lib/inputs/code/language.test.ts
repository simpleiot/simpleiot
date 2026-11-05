import { expect, test } from "vitest"
import { languageOf } from "./language.ts"

test("names the language from the file extension", () => {
	expect(languageOf("lamp.svg")).toBe("xml")
	expect(languageOf("10-groups.YAML")).toBe("yaml")
	expect(languageOf("config.json")).toBe("json")
	expect(languageOf("site.css")).toBe("css")
	expect(languageOf("tank.dbc")).toBeUndefined()
	expect(languageOf("")).toBeUndefined()
})
