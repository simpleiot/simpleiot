// editor makes a CodeMirror editor for a file's contents. It is imported
// on demand by CodeInput, so the editor's code is loaded only when a file
// is opened for editing, and nothing else in the UI depends on it.
import { css } from "@codemirror/lang-css"
import { javascript } from "@codemirror/lang-javascript"
import { json } from "@codemirror/lang-json"
import { xml } from "@codemirror/lang-xml"
import { yaml } from "@codemirror/lang-yaml"
import { indentWithTab } from "@codemirror/commands"
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language"
import { Compartment, EditorState, type Extension } from "@codemirror/state"
import { EditorView, keymap } from "@codemirror/view"
import { tags } from "@lezer/highlight"
import { basicSetup } from "codemirror"
import { languageOf } from "./language.ts"

export interface CodeEditor {
	// get is the editor's text
	get(): string
	// set replaces the text, for a change made elsewhere
	set(text: string): void
	// setName picks the highlighting from a file name
	setName(name: string): void
	destroy(): void
}

const languages = { xml, yaml, json, javascript, css }

function languageFor(name: string): Extension {
	const l = languageOf(name)
	return l ? languages[l]() : []
}

// the editor takes its colors from the page's theme tokens, so it follows
// the light and dark themes with the rest of the UI
const theme = EditorView.theme({
	"&": {
		backgroundColor: "var(--input-bg)",
		color: "var(--fg)",
		border: "1px solid var(--border)",
		borderRadius: "4px",
		fontSize: "13px",
		minHeight: "160px",
		maxHeight: "70vh",
	},
	"&.cm-focused": { outline: "2px solid var(--focus)" },
	".cm-scroller": { fontFamily: "var(--mono)", overflow: "auto" },
	".cm-content": { caretColor: "var(--fg)" },
	".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--fg)" },
	".cm-gutters": {
		backgroundColor: "var(--detail-bg)",
		color: "var(--muted)",
		borderRight: "1px solid var(--border)",
	},
	".cm-activeLine, .cm-activeLineGutter": {
		backgroundColor: "color-mix(in srgb, var(--fg) 5%, transparent)",
	},
	"&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, ::selection":
		{
			backgroundColor: "color-mix(in srgb, var(--blue) 25%, transparent)",
		},
	".cm-matchingBracket": { outline: "1px solid var(--muted)" },
})

const highlight = HighlightStyle.define([
	{ tag: [tags.tagName, tags.keyword, tags.operator], color: "var(--blue)" },
	{
		tag: [
			tags.attributeName,
			tags.propertyName,
			tags.definition(tags.propertyName),
			tags.className,
		],
		color: "var(--green)",
	},
	{ tag: [tags.string, tags.attributeValue], color: "var(--red)" },
	{ tag: [tags.number, tags.bool, tags.null, tags.atom], color: "var(--blue)" },
	{
		tag: [tags.comment, tags.meta, tags.processingInstruction],
		color: "var(--muted)",
		fontStyle: "italic",
	},
])

export function createCodeEditor(
	parent: HTMLElement,
	doc: string,
	name: string,
	onChange: (text: string) => void
): CodeEditor {
	const language = new Compartment()
	const view = new EditorView({
		parent,
		state: EditorState.create({
			doc,
			extensions: [
				basicSetup,
				keymap.of([indentWithTab]),
				theme,
				syntaxHighlighting(highlight),
				language.of(languageFor(name)),
				EditorView.updateListener.of((u) => {
					if (u.docChanged) {
						onChange(u.state.doc.toString())
					}
				}),
			],
		}),
	})
	return {
		get: () => view.state.doc.toString(),
		set: (text) => {
			view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } })
		},
		setName: (n) => {
			view.dispatch({ effects: language.reconfigure(languageFor(n)) })
		},
		destroy: () => view.destroy(),
	}
}
