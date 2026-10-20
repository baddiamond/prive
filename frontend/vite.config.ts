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
    // The app runs inside the OS webview (WebKit on macOS/Linux, WebView2 on Windows).
    // Safari 14 is the WebKit of macOS 11, the oldest macOS we support (docs/REQUIREMENTS.md).
    target: ["es2020", "safari14", "chrome100"],
  },
});
