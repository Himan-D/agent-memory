import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
  // tailwind.config.js is CommonJS — require() is intentional there.
  {
    files: ["tailwind.config.js"],
    rules: {
      "@typescript-eslint/no-require-imports": "off",
    },
  },
  // Downgrade prefer-const to a warning so pre-existing let declarations
  // don't break CI while we clean them up incrementally.
  {
    rules: {
      "prefer-const": "warn",
      "@typescript-eslint/no-explicit-any": "warn",
      // React Compiler rules are useful but too strict for common data-fetch
      // and navigation patterns used across the dashboard.
      "react-hooks/purity": "warn",
      "react-hooks/set-state-in-effect": "warn",
      "react-hooks/immutability": "warn",
    },
  },
]);

export default eslintConfig;
