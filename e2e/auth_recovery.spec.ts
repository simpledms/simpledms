import { expect, test } from "@playwright/test";

test("opens forgot-password flow without duplicating sign-in form", async ({ page }) => {
	await page.goto("/");
	await page.getByRole("link", { name: "Forget password?" }).first().click();

	await expect(page.getByRole("heading", { name: "Sign in | SimpleDMS" })).toHaveCount(1);
	await expect(page.getByRole("link", { name: "Powered by SimpleDMS" })).toHaveCount(1);
});

test("repeated forgot-password clicks stay idempotent", async ({ page }) => {
	test.fail(true, "Known issue: repeated clicks duplicate sign-in blocks");

	await page.goto("/");
	await page.getByRole("link", { name: "Forget password?" }).first().click();
	await page.getByRole("link", { name: "Forget password?" }).first().click();

	await expect(page.getByRole("heading", { name: "Sign in | SimpleDMS" })).toHaveCount(1);
	await expect(page.getByRole("link", { name: "Forget password?" })).toHaveCount(1);
});
