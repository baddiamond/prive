import { createRequire } from "node:module";
import type { Config } from "tailwindcss";

// Brand tokens (colors, radii, type scale, motion) generated from assets/brand/tokens/tokens.json.
// Components use only these semantic names, never raw colors (see docs/brand/BRAND.md).
const require = createRequire(import.meta.url);
const brandPreset = require("../assets/brand/tokens/tailwind.preset.js") as Partial<Config>;

export default {
  presets: [brandPreset],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
} satisfies Config;
