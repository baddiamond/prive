import { fileURLToPath } from "node:url";
import { defineConfig, mergeConfig, type Plugin } from "vite";
import base from "../vite.config";

// The UI regression harness: the real frontend with the Go bindings and the Wails runtime
// replaced by the in-memory stand-ins in e2e/mock, so it runs in a plain browser.
const mock = (file: string) => fileURLToPath(new URL(`./mock/${file}`, import.meta.url));

function wailsMock(): Plugin {
  return {
    name: "prive-wails-mock",
    enforce: "pre",
    resolveId(source) {
      if (source.endsWith("wailsjs/go/main/App")) return mock("App.js");
      if (source.endsWith("wailsjs/runtime/runtime")) return mock("runtime.js");
      return null;
    },
  };
}

export default mergeConfig(base, defineConfig({ root: fileURLToPath(new URL("..", import.meta.url)), plugins: [wailsMock()] }));
