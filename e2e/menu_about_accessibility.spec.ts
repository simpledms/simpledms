import { expect, test } from "@playwright/test";

import { expectVisibleMenuEntries, goToSpaces, goToUsers, signIn } from "./helpers";

test.describe("menu, about, and access control", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("shows core menu entries on dashboard, users, and spaces", async ({ page }) => {
		await page.goto("/dashboard/");
		await expectVisibleMenuEntries(page, ["dashboard Dashboard", "logout Sign out", "info About SimpleDMS"]);

		await goToUsers(page);
		await expectVisibleMenuEntries(page, ["dashboard Dashboard", "logout Sign out", "info About SimpleDMS"]);

		await goToSpaces(page);
		await expectVisibleMenuEntries(page, ["dashboard Dashboard", "logout Sign out", "info About SimpleDMS"]);
	});

	test("renders about page legal and attribution links", async ({ page }) => {
		await page.goto("/pages/about/");

		await expect(page).toHaveURL(/\/pages\/about\/$/);
		await expect(page.getByRole("heading", { name: "About SimpleDMS" })).toBeVisible();
		await expect(page.getByRole("link", { name: "simpledms.eu" })).toBeVisible();
		await expect(page.getByRole("link", { name: "simpledms.ch" })).toBeVisible();
		await expect(page.getByRole("link", { name: "https://github.com/simpledms/simpledms" })).toBeVisible();
		await expect(page.getByRole("heading", { name: "License", exact: true })).toBeVisible();
	});
});

test("shows Powered by SimpleDMS link on sign-in page", async ({ page }) => {
	await page.goto("/");
	await expect(page.getByRole("link", { name: "Powered by SimpleDMS" })).toBeVisible();
});

test("sign-out blocks protected routes", async ({ page }) => {
	await signIn(page);
	await page.request.post("/-/auth/sign-out-cmd");
	await page.goto("/");
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole("heading", { name: "Sign in | SimpleDMS" })).toBeVisible();

	await page.goto("/dashboard/");
	await expect(page).toHaveURL(/\/$/);
	await expect(page).toHaveTitle(/Sign in \| SimpleDMS/);
});
