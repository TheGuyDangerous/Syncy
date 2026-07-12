import { useEffect, useState, type FormEvent } from "react";
import { invoke } from "@tauri-apps/api/core";
import { getThemePref, setThemePref, type ThemePref } from "../lib/theme";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { StatusPill } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";

const SECTIONS = [
  { id: "general", label: "General" },
  { id: "appearance", label: "Appearance" },
  { id: "sync", label: "Sync" },
  { id: "network", label: "Network" },
  { id: "security", label: "Security" },
  { id: "integrations", label: "Integrations" },
  { id: "about", label: "About" },
] as const;

type SectionId = (typeof SECTIONS)[number]["id"];

const THEME_OPTIONS: { value: ThemePref; label: string }[] = [
  { value: "system", label: "System" },
  { value: "dark", label: "Dark" },
  { value: "light", label: "Light" },
];

interface AiProvider {
  id: string;
  name: string;
  baseUrl: string;
  apiKey: string;
  model: string;
  enabled: boolean;
}

const PROVIDERS_KEY = "syncy.ai-providers";

function loadProviders(): AiProvider[] {
  try {
    const raw = localStorage.getItem(PROVIDERS_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as AiProvider[]) : [];
  } catch {
    return [];
  }
}

function maskKey(key: string): string {
  return key.length > 4 ? `••••${key.slice(-4)}` : "••••";
}

function SectionHeader({ title, desc }: { title: string; desc: string }) {
  return (
    <header>
      <h2 className="settings-title">{title}</h2>
      <p className="settings-desc">{desc}</p>
    </header>
  );
}

function NoteCard({ name, hint }: { name: string; hint: string }) {
  return (
    <div className="card">
      <div className="setting-row">
        <div>
          <p className="setting-name">{name}</p>
          <p className="setting-hint">{hint}</p>
        </div>
      </div>
    </div>
  );
}

export default function Settings() {
  const [section, setSection] = useState<SectionId>("general");
  const [theme, setTheme] = useState<ThemePref>(getThemePref);
  const [providers, setProviders] = useState<AiProvider[]>(loadProviders);
  const [showProviderForm, setShowProviderForm] = useState(false);
  const [pName, setPName] = useState("");
  const [pUrl, setPUrl] = useState("");
  const [pKey, setPKey] = useState("");
  const [pModel, setPModel] = useState("");
  const [pError, setPError] = useState<string | null>(null);
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    invoke<string>("app_version")
      .then(setVersion)
      .catch(() => setVersion("dev"));
  }, []);

  function chooseTheme(pref: ThemePref) {
    setThemePref(pref);
    setTheme(pref);
  }

  function persist(next: AiProvider[]) {
    setProviders(next);
    try {
      localStorage.setItem(PROVIDERS_KEY, JSON.stringify(next));
    } catch {}
  }

  function addProvider(e: FormEvent) {
    e.preventDefault();
    const name = pName.trim();
    const baseUrl = pUrl.trim();
    if (!name || !baseUrl) {
      setPError("Name and base URL are required.");
      return;
    }
    persist([
      ...providers,
      {
        id: crypto.randomUUID(),
        name,
        baseUrl,
        apiKey: pKey.trim(),
        model: pModel.trim(),
        enabled: false,
      },
    ]);
    setPName("");
    setPUrl("");
    setPKey("");
    setPModel("");
    setPError(null);
    setShowProviderForm(false);
  }

  function toggleProvider(id: string) {
    persist(providers.map((p) => (p.id === id ? { ...p, enabled: !p.enabled } : p)));
  }

  function removeProvider(id: string) {
    persist(providers.filter((p) => p.id !== id));
  }

  return (
    <Screen title="Settings">
      <div className="settings">
        <nav className="settings-nav" aria-label="Settings sections">
          {SECTIONS.map((s) => (
            <button
              key={s.id}
              className="settings-nav-item"
              aria-current={section === s.id ? "true" : undefined}
              onClick={() => setSection(s.id)}
            >
              {s.label}
            </button>
          ))}
        </nav>

        <div className="settings-body">
          {section === "general" ? (
            <>
              <SectionHeader title="General" desc="Startup, notifications, and language." />
              <NoteCard
                name="Nothing to configure yet"
                hint="Startup, notification, and language options land here in a later release."
              />
            </>
          ) : null}

          {section === "appearance" ? (
            <>
              <SectionHeader title="Appearance" desc="How Syncy looks on this device." />
              <div className="card">
                <div className="setting-row">
                  <div>
                    <p className="setting-name">Theme</p>
                    <p className="setting-hint">Match the system, or pick one.</p>
                  </div>
                  <div className="segmented" role="group" aria-label="Theme">
                    {THEME_OPTIONS.map((opt) => (
                      <button
                        key={opt.value}
                        aria-pressed={theme === opt.value}
                        onClick={() => chooseTheme(opt.value)}
                      >
                        {opt.label}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </>
          ) : null}

          {section === "sync" ? (
            <>
              <SectionHeader title="Sync" desc="Defaults for how folders sync." />
              <NoteCard
                name="Engine defaults"
                hint="New folders send & receive and rescan automatically. Per-folder overrides land here later."
              />
            </>
          ) : null}

          {section === "network" ? (
            <>
              <SectionHeader title="Network" desc="How Syncy connects to your peers." />
              <NoteCard
                name="Automatic"
                hint="The engine picks listen addresses and relays for now. Manual control lands here later."
              />
            </>
          ) : null}

          {section === "security" ? (
            <>
              <SectionHeader title="Security" desc="Device trust and access." />
              <NoteCard
                name="Trusted devices only"
                hint="Only devices you pair can connect. Key management and access controls land here later."
              />
            </>
          ) : null}

          {section === "integrations" ? (
            <>
              <SectionHeader title="Integrations" desc="Optional services that extend Syncy." />
              <div className="card">
                <header className="card-head">
                  <div>
                    <h3 className="card-title">AI providers</h3>
                    <p className="setting-hint">
                      Optional. Stored locally on this device — never used unless you enable a
                      provider.
                    </p>
                  </div>
                  <button
                    className="btn btn--sm"
                    onClick={() => setShowProviderForm((v) => !v)}
                  >
                    <Icon name="plus" size={13} />
                    Add provider
                  </button>
                </header>

                {showProviderForm ? (
                  <form className="form-card form-card--divider" onSubmit={addProvider}>
                    <div className="form-grid">
                      <label className="field">
                        <span className="field-label">Name</span>
                        <input
                          className="input"
                          value={pName}
                          onChange={(e) => setPName(e.target.value)}
                          placeholder="My provider"
                        />
                      </label>
                      <label className="field">
                        <span className="field-label">Base URL</span>
                        <input
                          className="input input--mono"
                          value={pUrl}
                          onChange={(e) => setPUrl(e.target.value)}
                          placeholder="https://api.openai.com/v1"
                        />
                      </label>
                      <label className="field">
                        <span className="field-label">API key</span>
                        <input
                          className="input input--mono"
                          type="password"
                          value={pKey}
                          onChange={(e) => setPKey(e.target.value)}
                          placeholder="sk-…"
                        />
                      </label>
                      <label className="field">
                        <span className="field-label">Model</span>
                        <input
                          className="input input--mono"
                          value={pModel}
                          onChange={(e) => setPModel(e.target.value)}
                          placeholder="gpt-4o-mini"
                        />
                      </label>
                    </div>
                    {pError ? (
                      <p className="form-error" role="alert">
                        {pError}
                      </p>
                    ) : null}
                    <div className="form-actions">
                      <button type="submit" className="btn btn--primary">
                        Add provider
                      </button>
                      <button
                        type="button"
                        className="btn btn--ghost"
                        onClick={() => setShowProviderForm(false)}
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                ) : null}

                {providers.length === 0 && !showProviderForm ? (
                  <EmptyState
                    icon="sparkle"
                    title="No providers yet"
                    hint="Add one to use AI features when they arrive. Nothing runs until you enable it."
                  />
                ) : null}

                {providers.length > 0 ? (
                  <ul className="rows">
                    {providers.map((p) => (
                      <li key={p.id} className="row">
                        <label className="switch">
                          <input
                            type="checkbox"
                            checked={p.enabled}
                            onChange={() => toggleProvider(p.id)}
                            aria-label={`Enable ${p.name}`}
                          />
                          <span className="switch-track" />
                        </label>
                        <div className="row-body">
                          <p className="row-title">
                            {p.name}
                            {p.model ? (
                              <span className="pill">
                                <span className="mono">{p.model}</span>
                              </span>
                            ) : null}
                          </p>
                          <p className="row-sub mono">
                            {p.baseUrl}
                            {p.apiKey ? ` · ${maskKey(p.apiKey)}` : ""}
                          </p>
                        </div>
                        <div className="row-end">
                          <StatusPill
                            tone={p.enabled ? "ok" : "idle"}
                            label={p.enabled ? "Enabled" : "Off"}
                          />
                          <button
                            className="icon-btn icon-btn--danger"
                            aria-label={`Remove ${p.name}`}
                            onClick={() => removeProvider(p.id)}
                          >
                            <Icon name="trash" size={15} />
                          </button>
                        </div>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            </>
          ) : null}

          {section === "about" ? (
            <>
              <SectionHeader title="About" desc="Version and project info." />
              <div className="card about-card">
                <div className="brand-mark brand-mark--lg">S</div>
                <p className="about-name">Syncy</p>
                <p className="mono about-version">{version ? `v${version}` : "…"}</p>
                <p className="about-tag">Local-first, peer-to-peer folder sync.</p>
              </div>
            </>
          ) : null}
        </div>
      </div>
    </Screen>
  );
}
