import { expect, test, type Page } from "@playwright/test";
import { isSignInResponse, loginEmail, uniqueSuffix } from "./helpers";

async function submitSignIn(page: Page, email: string, password: string) {
	await page.getByRole("textbox", { name: "Email" }).fill(email);
	await page.getByRole("textbox", { name: "Password" }).fill(password);
	const responsePromise = page.waitForResponse(isSignInResponse);
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	return responsePromise;
}

test("rejects invalid credentials with safe error message", async ({ page }) => {
	// An unknown account keeps the shared admin account outside the sign-in rate limit, so
	// parallel tests can still sign in.
	await page.goto("/");
	await submitSignIn(page, `e2e-unknown-${uniqueSuffix()}@example.com`, "wrong-password");

	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole("heading", { name: "Sign in | SimpleDMS" })).toBeVisible();
	await expect(page.getByText("Invalid credentials. Please try again.")).toBeVisible();
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

// Tagged @state because it locks the shared admin account for sign-ins for 10 seconds.
test("rate-limits repeated failed logins", { tag: "@state" }, async ({ page }) => {
	await page.goto("/");

	await submitSignIn(page, loginEmail, "wrong-password-0");
	const response = await submitSignIn(page, loginEmail, "wrong-password-1");

	expect(response.status()).toBe(401);
	await expect(page.getByText(/Too many login attempts\. Please try again in \d+ seconds\./)).toBeVisible();
});
