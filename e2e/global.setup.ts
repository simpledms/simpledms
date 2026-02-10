import { chromium, expect, type FullConfig } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import path from "node:path";

async function globalSetup(config: FullConfig) {
	const baseURL = config.projects[0]?.use.baseURL as string | undefined;
	if (!baseURL) {
		throw new Error("Missing baseURL in Playwright config");
	}

	const email = process.env.E2E_LOGIN_EMAIL ?? "testing+admin@simpledms.app";
	const password = process.env.E2E_LOGIN_PASSWORD ?? "12345678";
	const storageStatePath = path.join(process.cwd(), "e2e/.auth/admin.json");

	await mkdir(path.dirname(storageStatePath), { recursive: true });

	const browser = await chromium.launch();
	const context = await browser.newContext({ ignoreHTTPSErrors: true });
	const page = await context.newPage();

	let isLoggedIn = false;

	for (let attempt = 0; attempt < 4; attempt++) {
		await page.goto(baseURL);
		await page.getByRole("textbox", { name: "Email" }).fill(email);
		await page.getByRole("textbox", { name: "Password" }).fill(password);
		await page.getByRole("button", { name: "Sign in" }).click();

		const reachedDashboard = await page
			.waitForURL(/\/dashboard\/$/, { timeout: 5_000 })
			.then(() => true)
			.catch(() => false);

		if (reachedDashboard) {
			isLoggedIn = true;
			break;
		}

		const rateLimited = await page.getByText(/Too many login attempts/i).isVisible().catch(() => false);
		if (rateLimited) {
			await page.waitForTimeout(11_000);
		}
	}

	await expect(isLoggedIn).toBeTruthy();

	await context.storageState({ path: storageStatePath });
	await browser.close();
}

export default globalSetup;
