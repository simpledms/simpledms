import { expect, test } from "@playwright/test";
import { loginEmail } from "./helpers";

test("rejects invalid credentials with safe error message", async ({ page }) => {
	await page.goto("/");

	for (let attempt = 0; attempt < 3; attempt++) {
		await page.getByRole("textbox", { name: "Email" }).fill(loginEmail);
		await page.getByRole("textbox", { name: "Password" }).fill(`wrong-password-${attempt}`);
		await page.getByRole("button", { name: "Sign in" }).click();

		const hasFeedback = await page
			.getByText(/Invalid credentials\. Please try again\.|Too many login attempts\./)
			.isVisible()
			.catch(() => false);
		if (hasFeedback) {
			break;
		}
	}

	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole("heading", { name: "Sign in | SimpleDMS" })).toBeVisible();
	await expect(page.getByText(/Invalid credentials\. Please try again\.|Too many login attempts\./)).toBeVisible();
});

test("supports temporary session checkbox toggle", async ({ page }) => {
	await page.goto("/");
	const temporarySession = page.getByRole("checkbox", { name: "Temporary session" });

	await expect(temporarySession).not.toBeChecked();
	await temporarySession.check();
	await expect(temporarySession).toBeChecked();
	await temporarySession.uncheck();
	await expect(temporarySession).not.toBeChecked();
});

test("rate-limits repeated failed logins", async ({ page }) => {
	await page.goto("/");

	for (let attempt = 0; attempt < 8; attempt++) {
		await page.getByRole("textbox", { name: "Email" }).fill(loginEmail);
		await page.getByRole("textbox", { name: "Password" }).fill(`wrong-password-${attempt}`);
		await page.getByRole("button", { name: "Sign in" }).click();

		const rateLimited = await page.getByText(/Too many login attempts\./).isVisible().catch(() => false);
		if (rateLimited) {
			break;
		}
	}

	await expect(page.getByText(/Too many login attempts\. Please try again in \d+ seconds\./)).toBeVisible();
});
