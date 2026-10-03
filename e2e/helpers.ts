import { expect, type Locator, type Page, type Response } from "@playwright/test";
import path from "node:path";

export const loginEmail = process.env.E2E_LOGIN_EMAIL ?? "dev+admin@simpledms.app";
export const loginPassword = process.env.E2E_LOGIN_PASSWORD ?? "12345678";

export function uniqueSuffix() {
	return `${Date.now()}-${Math.floor(Math.random() * 10_000)}`;
}

// The server accepts one password sign-in per account every 10 seconds.
const accountSignInIntervalMs = 10_000;

export function isSignInResponse(response: Response) {
	return response.request().method() === "POST"
		&& new URL(response.url()).pathname === "/-/auth/sign-in-cmd";
}

export async function signIn(page: Page) {
	for (let attempt = 0; attempt < 3; attempt++) {
		await page.goto("/");
		await page.getByRole("textbox", { name: "Email" }).fill(loginEmail);
		await page.getByRole("textbox", { name: "Password" }).fill(loginPassword);
		const responsePromise = page.waitForResponse(isSignInResponse);
		await page.getByRole("button", { name: "Sign in", exact: true }).click();
		const response = await responsePromise;

		if (response.headers()["hx-redirect"]) {
			await expect(page).toHaveURL(/\/dashboard\/$/);
			return;
		}

		const body = await response.text();
		if (response.status() !== 401 || !body.includes("Too many login attempts")) {
			throw new Error(
				`Sign-in failed with status ${response.status()}; check E2E_LOGIN_EMAIL and E2E_LOGIN_PASSWORD`,
			);
		}
		// Another sign-in for the same account happened recently, for example in global setup.
		await page.waitForTimeout(accountSignInIntervalMs);
	}

	throw new Error("Sign-in stayed rate-limited");
}

export async function goToSpaces(page: Page) {
	await page.goto("/dashboard/");
	const menu = page.getByRole("button", { name: "Open main menu", exact: true }).first();
	if (await menu.getAttribute("aria-expanded") !== "true") await menu.click();
	const spaces = page.getByRole("link", { name: "Spaces", exact: true }).first();
	if (!(await spaces.isVisible())) {
		await page.getByRole("button", { name: /^business / }).first().click();
	}
	await spaces.click();
	await expect(page).toHaveURL(/\/org\/[^/]+\/spaces\/$/);
}

export async function goToUsers(page: Page) {
	await page.goto("/dashboard/");
	await page.getByRole("link", { name: "Manage users" }).click();
	await expect(page).toHaveURL(/\/org\/[^/]+\/manage-users\/$/);
}

export async function openCreateSpaceDialog(page: Page) {
	const emptyStateCreate = page.getByRole("button", { name: "add Create space" });
	if (await emptyStateCreate.count()) {
		await emptyStateCreate.first().click();
	} else {
		await page.getByRole("link", { name: /Create space|^add$/ }).first().click();
	}
	await expect(page.getByRole("heading", { name: "Create space" })).toBeVisible();
}

export async function createSpaceAndSelect(page: Page, spaceName: string, documentTypes: string[] = []) {
	await goToSpaces(page);
	await openCreateSpaceDialog(page);
	await page.getByRole("textbox", { name: "Name" }).fill(spaceName);
	await page.getByRole("checkbox", { name: "Add me as space owner" }).check();
	for (const documentType of documentTypes) {
		await page.getByRole("checkbox", { name: documentType }).check();
	}
	await page.getByRole("button", { name: "Save" }).click();
	await expect(page.getByRole("heading", { name: spaceName })).toBeVisible();
	await page
		.getByRole("link")
		.filter({ has: page.getByRole("heading", { name: spaceName, exact: true }) })
		.click();
	await expect(page).toHaveURL(/\/space\/[^/]+\/browse\/$/);
}

export function fixturePath(fileName: string) {
	return path.join(process.cwd(), "e2e", "fixtures", fileName);
}

export async function uploadFileWithToolbar(page: Page, absoluteFilePath: string) {
	const menu = page.getByRole("button", { name: "Open main menu", exact: true }).first();
	if (await menu.getAttribute("aria-expanded") !== "true") await menu.click();
	await page.getByRole("link", { name: /Upload file/ }).first().click();
	const dialog = page.getByRole("dialog").filter({ hasText: "File upload" });
	await expect(dialog).toBeVisible();
	await dialog.locator("input[type=file]").first().setInputFiles(absoluteFilePath);
	await expect(dialog.getByText("Upload complete")).toBeVisible();
	await dialog.getByRole("button", { name: "close" }).click();
}

export async function openSpaceMenu(page: Page) {
	await page.getByRole("button", { name: "menu" }).click();
}

export async function openCreateUserDialog(page: Page) {
	await page.getByRole("link", { name: "add Add a new user" }).click();
	await expect(page.getByRole("heading", { name: "Create user" })).toBeVisible();
}

export async function expectVisibleMenuEntries(page: Page, entries: string[]) {
	await page.getByRole("button", { name: "menu" }).click();
	for (const entry of entries) {
		await expect(page.getByRole("link", { name: entry })).toBeVisible();
	}
}

export async function selectOptionByLabel(select: Locator, label: string) {
	await select.selectOption({ label });
}
