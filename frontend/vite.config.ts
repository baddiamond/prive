import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // The brand tokens live outside the frontend, in assets/brand/tokens.
    fs: { allow: [".."] },
  },
  build: {
    // The app runs inside the OS webview (WebKit on macOS/Linux, WebView2 on Windows),
    // so target engines that ship with still-supported OS versions.
    target: ["es2022", "safari15", "chrome100"],
  },
});
