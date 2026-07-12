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

function buildGo(out, extraEnv = {}) {
  execFileSync("go", ["build", "-trimpath", "-o", out, "./cmd/syncyd"], {
    cwd: engineDir,
    stdio: "inherit",
    env: { ...process.env, CGO_ENABLED: "0", ...extraEnv },
  });
  console.log(`built sidecar: ${out}`);
}

buildGo(join(outDir, `syncyd-${triple}${ext}`));

if (process.platform === "darwin") {
  const arm = join(outDir, "syncyd-aarch64-apple-darwin");
  const amd = join(outDir, "syncyd-x86_64-apple-darwin");
  buildGo(arm, { GOOS: "darwin", GOARCH: "arm64" });
  buildGo(amd, { GOOS: "darwin", GOARCH: "amd64" });
  const universal = join(outDir, "syncyd-universal-apple-darwin");
  execFileSync("lipo", ["-create", "-output", universal, arm, amd], { stdio: "inherit" });
  console.log(`built sidecar: ${universal}`);
}
