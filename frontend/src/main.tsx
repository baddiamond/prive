import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./i18n";
import "./fonts";
import "../../assets/brand/tokens/tokens.css";
import "./index.css";
import App from "./App";

const root = document.getElementById("root");
if (!root) throw new Error("missing #root element");

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
