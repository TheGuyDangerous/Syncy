import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const srcTauri = join(here, "..", "src-tauri");
const engineDir = join(here, "..", "..", "engine");

const rustc = execFileSync("rustc", ["-vV"], { encoding: "utf8" });
const triple = rustc.match(/host:\s*(\S+)/)?.[1];
if (!triple) {
  throw new Error("could not determine the Rust host target triple from rustc -vV");
}

const ext = process.platform === "win32" ? ".exe" : "";
const outDir = join(srcTauri, "binaries");
mkdirSync(outDir, { recursive: true });
const out = join(outDir, `syncyd-${triple}${ext}`);

execFileSync("go", ["build", "-trimpath", "-o", out, "./cmd/syncyd"], {
  cwd: engineDir,
  stdio: "inherit",
  env: { ...process.env, CGO_ENABLED: "0" },
});

console.log(`built sidecar: ${out}`);
