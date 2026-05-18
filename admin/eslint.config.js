import js from "@eslint/js";
import tseslint from "typescript-eslint";
import pluginVue from "eslint-plugin-vue";
import prettier from "eslint-config-prettier";
import autoImportGlobals from "./.eslintrc-auto-import.json" with { type: "json" };

export default [
  {
    ignores: [
      "dist/",
      "src/types/auto-imports.d.ts",
      "src/types/components.d.ts",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs["flat/recommended"],
  {
    languageOptions: {
      globals: autoImportGlobals.globals,
    },
  },
  {
    files: ["**/*.vue"],
    languageOptions: { parserOptions: { parser: tseslint.parser } },
  },
  prettier,
];
