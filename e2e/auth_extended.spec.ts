import { expect, test } from "@playwright/test";
import { loginEmail, loginPassword } from "./helpers";

test("supports keyboard flow on sign in form", async ({ page }) => {
	await page.goto("/");
	await page.getByRole("textbox", { name: "Email" }).fill(loginEmail);
	await page.keyboard.press("Tab");
	await expect(page.getByRole("textbox", { name: "Password" })).toBeFocused();
	await page.keyboard.type(loginPassword);
	await expect(page.getByRole("textbox", { name: "Password" })).toHaveValue(loginPassword);
	await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
});

test.describe("authenticated extended", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("shows sign out action in main menu", async ({ page }) => {
		await page.goto("/dashboard/");
		await page.getByRole("button", { name: "menu" }).click();
		await expect(page.getByRole("link", { name: "logout Sign out" })).toBeVisible();
	});

	test("opens passphrase dialog only once", async ({ page }) => {
		await page.goto("/dashboard/");
		await page.getByRole("link", { name: "Set passphrase" }).click();
		await expect(page.getByRole("heading", { name: "Change passphrase" })).toHaveCount(1);
		await expect(page.getByRole("textbox", { name: "Confirm new passphrase" })).toHaveCount(1);
	});
});

test.describe("mobile viewport", () => {
	test.use({
		storageState: "e2e/.auth/admin.json",
		viewport: { width: 390, height: 844 },
	});

	test("shows dashboard and menu controls on mobile", async ({ page }) => {
		await page.goto("/dashboard/");
		await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
		await expect(page.getByRole("button", { name: "menu" })).toBeVisible();
		await page.getByRole("button", { name: "menu" }).click();
		await expect(page.getByRole("link", { name: /Spaces «/ })).toBeVisible();
	});
});
