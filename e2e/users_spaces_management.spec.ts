import { expect, test } from "@playwright/test";

import { goToSpaces, goToUsers, loginEmail, openCreateSpaceDialog, openCreateUserDialog, selectOptionByLabel, uniqueSuffix } from "./helpers";

test.describe("users and spaces management", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("renders manage-users page with add action and current user", async ({ page }) => {
		await goToUsers(page);
		await expect(page.getByRole("heading", { name: /Users «/ })).toBeVisible();
		await expect(page.getByRole("link", { name: "add Add a new user" })).toBeVisible();
		await expect(page.getByText(loginEmail)).toBeVisible();
	});

	test("rejects invalid email in create-user form", async ({ page }) => {
		await goToUsers(page);
		await openCreateUserDialog(page);

		const emailInput = page.getByRole("textbox", { name: "Email" });
		await emailInput.fill("not-an-email");
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByRole("heading", { name: "Create user" })).toBeVisible();
		await expect(emailInput).toHaveValue("not-an-email");
		expect(await emailInput.evaluate((element) => (element as HTMLInputElement).checkValidity())).toBeFalsy();
	});

	test("creates user with unique email and selected role", async ({ page }) => {
		const suffix = uniqueSuffix();
		const userEmail = `playwright.user.${suffix}@example.com`;

		await goToUsers(page);
		await openCreateUserDialog(page);
		await selectOptionByLabel(page.getByRole("combobox", { name: "Role" }), "Owner");
		await page.getByRole("textbox", { name: "Email" }).fill(userEmail);
		await page.getByRole("textbox", { name: "First name" }).fill("Playwright");
		await page.getByRole("textbox", { name: "Last name" }).fill("Owner");
		await selectOptionByLabel(page.getByRole("combobox", { name: "Language" }), "English");
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByText(userEmail)).toBeVisible();
	});

	test("prevents creating duplicate user email", async ({ page }) => {
		await goToUsers(page);
		await openCreateUserDialog(page);
		await page.getByRole("textbox", { name: "Email" }).fill(loginEmail);
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByRole("heading", { name: "Create user" })).toBeVisible();
		await expect(page.getByText(loginEmail)).toHaveCount(1);
	});

	test("requires name in create-space form", async ({ page }) => {
		await goToSpaces(page);
		await openCreateSpaceDialog(page);
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByRole("heading", { name: "Create space" })).toBeVisible();
		await expect(page.getByRole("textbox", { name: "Name" })).toBeFocused();
	});

	test("creates a space with selected document types", async ({ page }) => {
		const spaceName = `E2E Space ${uniqueSuffix()}`;

		await goToSpaces(page);
		const selectCountBefore = await page.getByRole("link", { name: "Select" }).count();
		await openCreateSpaceDialog(page);
		await page.getByRole("textbox", { name: "Name" }).fill(spaceName);
		await page.getByRole("checkbox", { name: "Add me as space owner" }).check();
		await page.getByRole("checkbox", { name: "Invoice" }).check();
		await page.getByRole("checkbox", { name: "Receipt" }).check();
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByRole("heading", { name: spaceName })).toBeVisible();
		await expect(page.getByRole("link", { name: "Select" })).toHaveCount(selectCountBefore + 1);
		await expect(page.getByRole("heading", { name: "No spaces available yet." })).toHaveCount(0);
		const headings = page.getByRole("heading", { level: 3 });
		let headingIndex = -1;
		for (let i = 0; i < (await headings.count()); i++) {
			const text = (await headings.nth(i).innerText()).trim();
			if (text === spaceName) {
				headingIndex = i;
				break;
			}
		}
		expect(headingIndex).toBeGreaterThanOrEqual(0);
		await page.getByRole("link", { name: "Select" }).nth(headingIndex).click();
		await expect(page).toHaveURL(/\/space\/[^/]+\/browse\/$/);
		await page.goto(page.url().replace(/\/browse\/$/, "/document-types/"));

		await expect(page).toHaveURL(/\/space\/[^/]+\/document-types\/$/);
		await expect(page.getByRole("heading", { name: "Invoice" })).toBeVisible();
		await expect(page.getByRole("heading", { name: "Receipt" })).toBeVisible();
		await expect(page.getByRole("heading", { name: "Contract" })).toHaveCount(0);
	});

	test("allows creating a space with owner toggle off", async ({ page }) => {
		await goToSpaces(page);
		const selectCountBefore = await page.getByRole("link", { name: "Select" }).count();
		await openCreateSpaceDialog(page);
		await page.getByRole("textbox", { name: "Name" }).fill(`No-owner ${uniqueSuffix()}`);
		const ownerToggle = page.getByRole("checkbox", { name: "Add me as space owner" });
		await ownerToggle.uncheck();
		await expect(ownerToggle).not.toBeChecked();
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByRole("link", { name: "Select" })).toHaveCount(selectCountBefore + 1);
	});
});
