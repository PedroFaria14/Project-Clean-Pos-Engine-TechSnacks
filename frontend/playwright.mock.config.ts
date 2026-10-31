import { defineConfig } from "@playwright/test";

export default defineConfig({
  forbidOnly: !!process.env.CI,
  testDir: "./e2e/mock", workers: 1, retries: 0, timeout: 20_000,
  outputDir: "test-results/mock",
  reporter: [["list"], ["html", { outputFolder: "playwright-report/mock", open: "never" }]],
  use: { baseURL: "http://127.0.0.1:4173", browserName: "chromium", channel: process.env.PLAYWRIGHT_CHANNEL || undefined, trace: "off", screenshot: "only-on-failure", viewport: { width: 1280, height: 800 } },
  webServer: { command: "npm run dev -- --host 127.0.0.1 --port 4173 --strictPort", url: "http://127.0.0.1:4173", reuseExistingServer: false, timeout: 30_000 },
});
