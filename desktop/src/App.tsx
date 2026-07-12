import { useEffect, useState } from "react";
import { invoke } from "@tauri-apps/api/core";

export default function App() {
  const [version, setVersion] = useState("");

  useEffect(() => {
    invoke<string>("app_version")
      .then(setVersion)
      .catch(() => setVersion("dev"));
  }, []);

  return (
    <main className="app">
      <div className="ambient" aria-hidden />
      <section className="hero">
        <div className="mark">S</div>
        <h1 className="wordmark">Syncy</h1>
        <p className="tagline">Local-first, peer-to-peer folder sync</p>
        <div className="status">
          <span className="dot" />
          Engine idle
        </div>
      </section>
      <footer className="foot">
        <span>{version ? `v${version}` : "…"}</span>
        <span className="sep">•</span>
        <span>early preview</span>
      </footer>
    </main>
  );
}
