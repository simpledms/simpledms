import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
	testDir: "./e2e",
	globalSetup: "./e2e/global.setup.ts",
	timeout: 30_000,
	expect: {
		timeout: 5_000,
	},
	fullyParallel: false,
	workers: 1,
	retries: 0,
	reporter: "list",
	use: {
		baseURL: process.env.E2E_BASE_URL ?? "https://localhost:7003",
		ignoreHTTPSErrors: true,
		trace: "on-first-retry",
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],
});
