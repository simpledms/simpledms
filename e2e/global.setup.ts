import { chromium, type FullConfig } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import path from "node:path";

import { signIn } from "./helpers";

async function globalSetup(config: FullConfig) {
	const baseURL = config.projects[0]?.use.baseURL as string | undefined;
	if (!baseURL) {
		throw new Error("Missing baseURL in Playwright config");
	}

	const storageStatePath = path.join(process.cwd(), "e2e/.auth/admin.json");

	await mkdir(path.dirname(storageStatePath), { recursive: true });

	const browser = await chromium.launch();
	const context = await browser.newContext({ baseURL, ignoreHTTPSErrors: true });
	const page = await context.newPage();

	await signIn(page);

	await context.storageState({ path: storageStatePath });
	await browser.close();
}

export default globalSetup;
