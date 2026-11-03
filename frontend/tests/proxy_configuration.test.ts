import assert from "node:assert/strict";
import { test } from "node:test";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const shell = process.env.TEST_SH ?? (process.platform === "win32" ? undefined : "sh");
const script = fileURLToPath(new URL("../docker/start_frontend.sh", import.meta.url)).replace(/\\/g, "/");

test("runtime proxy rejects unsafe ports, malformed origins and self-proxy loops before writing a config", { skip: shell ? false : "requires a POSIX shell; set TEST_SH to Git Bash on Windows" }, () => {
  for (const overrides of [
    { PORT: "0" }, { PORT: "65536" }, { PORT: "${PORT}" },
    { API_PROXY_TARGET: "https://api.example/path" },
    { API_PROXY_TARGET: "https://user:password@api.example" },
    { API_PROXY_TARGET: "https://api.example?token=anything" },
    { API_PROXY_TARGET: "https://api.example\nmalformed" },
    { API_PROXY_TARGET: "http://API.ONRENDER.COM" },
    { API_PROXY_TARGET: "https://self.onrender.com", RENDER_EXTERNAL_HOSTNAME: "self.onrender.com" },
  ]) {
    const result = spawnSync(shell!, [script], { encoding: "utf8", env: { ...process.env, PORT: "8080", API_PROXY_TARGET: "http://backend:8080", ...overrides } });
    assert.equal(result.status, 1, `runtime accepted invalid configuration or failed unexpectedly: ${result.stderr}`);
    assert.ok(!result.stdout.includes("password"));
  }
});
