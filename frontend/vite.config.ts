import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    // The app runs inside the OS webview (WebKit on macOS/Linux, WebView2 on Windows),
    // so target engines that ship with still-supported OS versions.
    target: ["es2022", "safari15", "chrome100"],
  },
});
