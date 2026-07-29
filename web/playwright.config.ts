import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  fullyParallel: false,
  use: {
    baseURL: "http://127.0.0.1:4174",
    channel: "chrome",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "go run ./e2e/server",
    url: "http://127.0.0.1:4174/healthz",
    reuseExistingServer: false,
    timeout: 60_000,
  },
});
