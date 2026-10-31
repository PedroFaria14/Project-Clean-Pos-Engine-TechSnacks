import { defineConfig } from "@playwright/test";
import { validateTestEnvironment } from "./e2e/real/database.mjs";

validateTestEnvironment();
export default defineConfig({
  forbidOnly: !!process.env.CI,
  testDir: "./e2e/real", workers: 1, retries: 0, timeout: 30_000,
  outputDir: "test-results/real",
  reporter: [["list"], ["html", { outputFolder: "playwright-report/real", open: "never" }]],
  use: { baseURL: process.env.E2E_BASE_URL, browserName: "chromium", channel: process.env.PLAYWRIGHT_CHANNEL || undefined, trace: "off", screenshot: "only-on-failure", viewport: { width: 1280, height: 900 } },
});
