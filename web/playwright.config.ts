import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  use: {
    baseURL: "http://127.0.0.1:5173",
    viewport: { width: 1500, height: 1000 },
    launchOptions: {
      executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    },
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command: "go run ./cmd/server",
      cwd: "..",
      url: "http://127.0.0.1:8080/api/v1/healthz",
      env: {
        AUTH_TOKEN: "playwright-token",
        GENERATION_PROVIDER: "template",
        PORT: "127.0.0.1:8080",
        MONGO_URI: "",
      },
    },
    {
      command: "npm run dev -- --host 127.0.0.1",
      url: "http://127.0.0.1:5173",
    },
  ],
});
