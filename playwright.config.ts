import { defineConfig, devices } from "@playwright/test";

// Tests tagged @state change the shared account or app state (for example password or
// passphrase), so they run alone after all other tests have finished.
const stateTag = /@state/;

export default defineConfig({
	testDir: "./e2e",
	globalSetup: "./e2e/global.setup.ts",
	timeout: 30_000,
	expect: {
		timeout: 5_000,
	},
	// Tests create their own uniquely named spaces and files, so they can run in parallel.
	fullyParallel: true,
	// 25% keeps the dev server and SQLite responsive (8 workers on a 32-core machine).
	workers: process.env.CI ? 2 : "25%",
	retries: 0,
	reporter: "list",
	use: {
		baseURL: process.env.E2E_BASE_URL ?? "https://localhost:7003",
		ignoreHTTPSErrors: true,
		// Fail broken locators fast instead of waiting for the whole test timeout.
		actionTimeout: 10_000,
		navigationTimeout: 15_000,
		trace: "on-first-retry",
	},
	projects: [
		{
			name: "chromium",
			grepInvert: stateTag,
			use: { ...devices["Desktop Chrome"] },
		},
		{
			name: "chromium-state",
			grep: stateTag,
			dependencies: ["chromium"],
			workers: 1,
			use: { ...devices["Desktop Chrome"] },
		},
	],
});
