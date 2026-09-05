import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import "@warehouse/ui-kit/tokens.css";
import App from "./App";

/** Standalone dev entry -- lets facility-mfe run and be worked on in
 *  isolation (npm run dev, :5186) without the shell attached. The shell
 *  imports App.tsx directly via Module Federation and provides its own
 *  BrowserRouter + AppShell chrome; this file is dev-only scaffolding. */
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <div style={{ padding: "var(--wh-space-5)" }}>
        <App />
      </div>
    </BrowserRouter>
  </StrictMode>,
);
