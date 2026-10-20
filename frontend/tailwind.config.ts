import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  // The "dark" class on <html> is set by src/lib/theme.ts (system, light or dark).
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        sans: [
          "system-ui",
          "-apple-system",
          "BlinkMacSystemFont",
          '"Segoe UI"',
          "Roboto",
          "Ubuntu",
          "sans-serif",
        ],
      },
    },
  },
  plugins: [],
} satisfies Config;
