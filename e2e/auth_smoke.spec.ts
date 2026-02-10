import { expect, test } from "@playwright/test";

test("redirects unauthenticated users to sign in", async ({ page }) => {
	await page.goto("/dashboard/");
	await expect(page).toHaveTitle(/Sign in \| SimpleDMS/);
	await expect(page.getByRole("heading", { name: "Sign in | SimpleDMS" })).toBeVisible();
});

test.describe("authenticated smoke", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("shows dashboard tasks", async ({ page }) => {
		await page.goto("/dashboard/");
		await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
		await expect(page.getByRole("heading", { name: "Open tasks" })).toBeVisible();
		await expect(page.getByRole("link", { name: "Manage spaces" }).first()).toBeVisible();
		await expect(page.getByRole("link", { name: "Manage users" })).toBeVisible();
		await expect(page.getByRole("link", { name: "Set passphrase" })).toBeVisible();
	});

	test("opens and closes set password dialog", async ({ page }) => {
		await page.goto("/dashboard/");
		await page.getByRole("link", { name: "Set password now" }).click();
		await expect(page.getByRole("heading", { name: "Set password" })).toBeVisible();
		await expect(page.getByRole("textbox", { name: "New password" })).toBeVisible();
		await expect(page.getByRole("textbox", { name: "Confirm password" })).toBeVisible();
		await page.getByRole("button", { name: "Close" }).click();
		await expect(page.getByRole("heading", { name: "Set password" })).toBeHidden();
	});

	test("opens and closes passphrase dialog", async ({ page }) => {
		await page.goto("/dashboard/");
		await page.getByRole("link", { name: "Set passphrase" }).click();
		await expect(page.getByRole("heading", { name: "Change passphrase" })).toBeVisible();
		await expect(page.getByRole("textbox", { name: "Current passphrase (optional)" })).toBeVisible();
		await expect(page.getByRole("textbox", { name: "New passphrase", exact: true })).toBeVisible();
		await expect(page.getByRole("textbox", { name: "Confirm new passphrase" })).toBeVisible();
		await page.getByRole("button", { name: "Close" }).click();
		await expect(page.getByRole("heading", { name: "Change passphrase" })).toBeHidden();
	});

	test("navigates to spaces page and shows empty state", async ({ page }) => {
		await page.goto("/dashboard/");
		await page.getByRole("button", { name: "menu" }).click();
		await page.getByRole("link", { name: /Spaces «/ }).click();
		await expect(page).toHaveURL(/\/org\/[^/]+\/spaces\/$/);
		await expect(page.getByRole("heading", { name: /Spaces/ })).toBeVisible();
		await expect(page.getByRole("link", { name: "add", exact: true })).toBeVisible();
	});
});
