import js from "@eslint/js"
import prettier from "eslint-config-prettier"
import svelte from "eslint-plugin-svelte"
import globals from "globals"
import ts from "typescript-eslint"

export default ts.config(
	{ ignores: ["build/", ".svelte-kit/", "node_modules/", "test-results/", "playwright-report/"] },
	js.configs.recommended,
	...ts.configs.strictTypeChecked,
	...svelte.configs.recommended,
	prettier,
	...svelte.configs.prettier,
	{
		languageOptions: {
			globals: { ...globals.browser, ...globals.node },
			parserOptions: {
				projectService: true,
				extraFileExtensions: [".svelte"],
				tsconfigRootDir: import.meta.dirname,
			},
		},
		rules: {
			"@typescript-eslint/no-explicit-any": "error",
			"@typescript-eslint/restrict-template-expressions": ["error", { allowNumber: true }],
			"@typescript-eslint/no-confusing-void-expression": "off",
			"@typescript-eslint/no-floating-promises": ["error", { ignoreVoid: true }],
			eqeqeq: ["error", "always"],
			"no-var": "error",
			"prefer-const": ["error", { destructuring: "all" }],
			"object-shorthand": ["warn", "always"],
		},
	},
	{
		files: ["**/*.svelte", "**/*.svelte.ts"],
		languageOptions: {
			parserOptions: { parser: ts.parser },
		},
		rules: {
			// svelte/prefer-const knows $props() destructuring is reassigned
			"prefer-const": "off",
			"svelte/prefer-const": ["error", { destructuring: "all" }],
		},
	},
	{
		files: ["*.config.js", "*.config.ts"],
		...ts.configs.disableTypeChecked,
	}
)
