import { useEffect, useState, type FormEvent } from "react";
import { invoke } from "@tauri-apps/api/core";
import { enable, disable, isEnabled } from "@tauri-apps/plugin-autostart";
import { getThemePref, setThemePref, type ThemePref } from "../lib/theme";
import {
  api,
  errorMessage,
  type AiKind,
  type AiConfigInput,
  type DiscoverySettings,
} from "../lib/api";
import { Screen } from "../components/Screen";
import { Select } from "../components/Select";
import { BrandMark } from "../components/BrandMark";

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

const AI_PROVIDERS: { value: AiKind; label: string; defaultUrl: string; needsKey: boolean }[] = [
  { value: "openai", label: "OpenAI", defaultUrl: "https://api.openai.com/v1", needsKey: true },
  { value: "anthropic", label: "Anthropic", defaultUrl: "https://api.anthropic.com/v1", needsKey: true },
  {
    value: "gemini",
    label: "Google Gemini",
    defaultUrl: "https://generativelanguage.googleapis.com/v1beta",
    needsKey: true,
  },
  { value: "openrouter", label: "OpenRouter", defaultUrl: "https://openrouter.ai/api/v1", needsKey: true },
  { value: "ollama", label: "Ollama (local)", defaultUrl: "http://localhost:11434/v1", needsKey: false },
  { value: "lmstudio", label: "LM Studio (local)", defaultUrl: "http://localhost:1234/v1", needsKey: false },
  { value: "custom", label: "Custom (OpenAI-compatible)", defaultUrl: "", needsKey: false },
];

function providerMeta(kind: AiKind) {
  return AI_PROVIDERS.find((p) => p.value === kind) ?? AI_PROVIDERS[0];
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
  const [version, setVersion] = useState<string | null>(null);
  const [autostart, setAutostart] = useState(false);

  const [aiEnabled, setAiEnabled] = useState(false);
  const [aiKind, setAiKind] = useState<AiKind>("openai");
  const [aiBaseUrl, setAiBaseUrl] = useState("");
  const [aiModel, setAiModel] = useState("");
  const [aiKey, setAiKey] = useState("");
  const [aiHasKey, setAiHasKey] = useState(false);
  const [aiBusy, setAiBusy] = useState<null | "save" | "test">(null);
  const [aiMsg, setAiMsg] = useState<{ ok: boolean; text: string } | null>(null);

  const [discovery, setDiscovery] = useState<DiscoverySettings | null>(null);
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);

  useEffect(() => {
    invoke<string>("app_version")
      .then(setVersion)
      .catch(() => setVersion("dev"));
    isEnabled()
      .then(setAutostart)
      .catch(() => setAutostart(false));
    api
      .aiConfig()
      .then((cfg) => {
        setAiEnabled(cfg.enabled);
        setAiKind(cfg.kind || "openai");
        setAiBaseUrl(cfg.base_url);
        setAiModel(cfg.model);
        setAiHasKey(cfg.has_api_key);
      })
      .catch(() => {});
    api
      .discovery()
      .then(setDiscovery)
      .catch((e: unknown) => setDiscoveryError(errorMessage(e)));
  }, []);

  function toggleDiscovery(key: keyof DiscoverySettings) {
    if (!discovery) return;
    const prev = discovery;
    const next = { ...discovery, [key]: !discovery[key] };
    setDiscovery(next);
    setDiscoveryError(null);
    api
      .saveDiscovery(next)
      .then(setDiscovery)
      .catch((e: unknown) => {
        setDiscovery(prev);
        setDiscoveryError(errorMessage(e));
      });
  }

  function chooseTheme(pref: ThemePref) {
    setThemePref(pref);
    setTheme(pref);
  }

  async function toggleAutostart() {
    try {
      if (autostart) {
        await disable();
        setAutostart(false);
      } else {
        await enable();
        setAutostart(true);
      }
    } catch {
      setAutostart((v) => v);
    }
  }

  function aiInput(): AiConfigInput {
    return {
      enabled: aiEnabled,
      kind: aiKind,
      base_url: aiBaseUrl.trim(),
      model: aiModel.trim(),
      api_key: aiKey,
    };
  }

  async function saveAi(e: FormEvent) {
    e.preventDefault();
    setAiBusy("save");
    setAiMsg(null);
    try {
      const view = await api.saveAiConfig(aiInput());
      setAiHasKey(view.has_api_key);
      setAiKey("");
      setAiMsg({ ok: true, text: "Saved." });
    } catch (err) {
      setAiMsg({ ok: false, text: errorMessage(err) });
    } finally {
      setAiBusy(null);
    }
  }

  async function testAi() {
    setAiBusy("test");
    setAiMsg(null);
    try {
      await api.testAi(aiInput());
      setAiMsg({ ok: true, text: "Connected — the provider replied." });
    } catch (err) {
      setAiMsg({ ok: false, text: errorMessage(err) });
    } finally {
      setAiBusy(null);
    }
  }

  const meta = providerMeta(aiKind);

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
              <SectionHeader title="General" desc="Startup and background behavior." />
              <div className="card">
                <div className="setting-row">
                  <div>
                    <p className="setting-name">Start Syncy at login</p>
                    <p className="setting-hint">
                      Launch in the background when you sign in, so your folders keep syncing.
                    </p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={autostart}
                      onChange={toggleAutostart}
                      aria-label="Start Syncy at login"
                    />
                    <span className="switch-track" />
                  </label>
                </div>
              </div>
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
              <SectionHeader title="Network" desc="How Syncy finds and reaches your friends." />
              <div className="card">
                <div className="setting-row">
                  <div>
                    <p className="setting-name">Local network</p>
                    <p className="setting-hint">Find devices on the same Wi-Fi automatically.</p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={discovery?.local ?? true}
                      disabled={!discovery}
                      onChange={() => toggleDiscovery("local")}
                      aria-label="Local network discovery"
                    />
                    <span className="switch-track" />
                  </label>
                </div>
                <div className="setting-row">
                  <div>
                    <p className="setting-name">
                      Internet <span className="pill">Alpha</span>
                    </p>
                    <p className="setting-hint">
                      Reach friends on other networks. Works when your router supports UPnP or
                      you've forwarded the port; both sides behind carrier NAT isn't supported
                      yet.
                    </p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={discovery?.internet ?? false}
                      disabled={!discovery}
                      onChange={() => toggleDiscovery("internet")}
                      aria-label="Internet discovery"
                    />
                    <span className="switch-track" />
                  </label>
                </div>
                {discoveryError ? (
                  <div className="setting-row">
                    <p className="form-error" role="alert">
                      {discoveryError}
                    </p>
                  </div>
                ) : null}
              </div>
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
                <div className="setting-row">
                  <div>
                    <p className="setting-name">AI assistant</p>
                    <p className="setting-hint">
                      Bring your own provider to explain conflicts and summarize logs. Off by
                      default; your key stays on this device and is used only when enabled.
                    </p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={aiEnabled}
                      onChange={(e) => setAiEnabled(e.target.checked)}
                      aria-label="Enable AI assistant"
                    />
                    <span className="switch-track" />
                  </label>
                </div>

                <form className="form-card form-card--divider" onSubmit={saveAi}>
                  <div className="form-grid">
                    <div className="field">
                      <span className="field-label">Provider</span>
                      <Select
                        value={aiKind}
                        onChange={(v) => setAiKind(v as AiKind)}
                        options={AI_PROVIDERS.map((p) => ({ value: p.value, label: p.label }))}
                        ariaLabel="Provider"
                      />
                    </div>
                    <label className="field">
                      <span className="field-label">Model</span>
                      <input
                        className="input input--mono"
                        value={aiModel}
                        onChange={(e) => setAiModel(e.target.value)}
                        placeholder="gpt-4o-mini"
                      />
                    </label>
                    <label className="field">
                      <span className="field-label">
                        Base URL{meta.defaultUrl ? " (optional)" : ""}
                      </span>
                      <input
                        className="input input--mono"
                        value={aiBaseUrl}
                        onChange={(e) => setAiBaseUrl(e.target.value)}
                        placeholder={meta.defaultUrl || "https://your-endpoint/v1"}
                      />
                    </label>
                    <label className="field">
                      <span className="field-label">
                        API key{aiHasKey ? " · saved" : meta.needsKey ? "" : " (not required)"}
                      </span>
                      <input
                        className="input input--mono"
                        type="password"
                        value={aiKey}
                        onChange={(e) => setAiKey(e.target.value)}
                        placeholder={
                          aiHasKey
                            ? "•••••••• — leave blank to keep"
                            : meta.needsKey
                              ? "sk-…"
                              : "not needed for local providers"
                        }
                      />
                    </label>
                  </div>
                  {aiMsg ? (
                    <p
                      className={aiMsg.ok ? "form-ok" : "form-error"}
                      role={aiMsg.ok ? "status" : "alert"}
                    >
                      {aiMsg.text}
                    </p>
                  ) : null}
                  <div className="form-actions">
                    <button type="submit" className="btn btn--primary" disabled={aiBusy !== null}>
                      {aiBusy === "save" ? "Saving…" : "Save"}
                    </button>
                    <button
                      type="button"
                      className="btn btn--ghost"
                      onClick={testAi}
                      disabled={aiBusy !== null}
                    >
                      {aiBusy === "test" ? "Testing…" : "Test connection"}
                    </button>
                  </div>
                </form>
              </div>
            </>
          ) : null}

          {section === "about" ? (
            <>
              <SectionHeader title="About" desc="Version and project info." />
              <div className="card about-card">
                <BrandMark size={44} />
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
