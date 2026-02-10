import { expect, type Locator, type Page } from "@playwright/test";
import path from "node:path";

export const loginEmail = process.env.E2E_LOGIN_EMAIL ?? "testing+admin@simpledms.app";
export const loginPassword = process.env.E2E_LOGIN_PASSWORD ?? "12345678";

export function uniqueSuffix() {
	return `${Date.now()}-${Math.floor(Math.random() * 10_000)}`;
}

export async function signIn(page: Page) {
	for (let attempt = 0; attempt < 4; attempt++) {
		await page.goto("/");
		await page.getByRole("textbox", { name: "Email" }).fill(loginEmail);
		await page.getByRole("textbox", { name: "Password" }).fill(loginPassword);
		await page.getByRole("button", { name: "Sign in" }).click();

		const reachedDashboard = await page
			.waitForURL(/\/dashboard\/$/, { timeout: 5_000 })
			.then(() => true)
			.catch(() => false);
		if (reachedDashboard) {
			return;
		}

		const rateLimited = await page.getByText(/Too many login attempts/i).isVisible().catch(() => false);
		if (rateLimited) {
			await page.waitForTimeout(11_000);
		}
	}

	await expect(page).toHaveURL(/\/dashboard\/$/);
}

export async function goToSpaces(page: Page) {
	await page.goto("/dashboard/");
	await page.getByRole("link", { name: "Manage spaces" }).first().click();
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
		await page.getByRole("link", { name: "add", exact: true }).click();
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
	const headings = page.getByRole("heading", { level: 3 });
	let headingIndex = -1;
	for (let i = 0; i < (await headings.count()); i++) {
		const text = (await headings.nth(i).innerText()).trim();
		if (text === spaceName) {
			headingIndex = i;
			break;
		}
	}
	if (headingIndex < 0) {
		throw new Error(`Could not find space card: ${spaceName}`);
	}
	await page.getByRole("link", { name: "Select" }).nth(headingIndex).click();
	await expect(page).toHaveURL(/\/space\/[^/]+\/browse\/$/);
}

export function fixturePath(fileName: string) {
	return path.join(process.cwd(), "e2e", "fixtures", fileName);
}

export async function uploadFileWithToolbar(page: Page, absoluteFilePath: string) {
	await page.getByRole("link", { name: "upload_file", exact: true }).click();
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
