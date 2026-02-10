import { expect, test } from "@playwright/test";

const allowStateMutation = process.env.E2E_ALLOW_STATE_MUTATION === "1";

test.describe("password and passphrase", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("shows mismatch validation for set password", async ({ page }) => {
		await page.goto("/dashboard/");
		const setPasswordLink = page.getByRole("link", { name: "Set password now" });
		test.skip(!(await setPasswordLink.count()), "Set-password task only appears for temporary-password accounts");

		await setPasswordLink.click();
		await page.getByRole("textbox", { name: "New password" }).fill("abc12345");
		await page.getByRole("textbox", { name: "Confirm password" }).fill("abc12346");
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByText("Passwords do not match.")).toBeVisible();
		await expect(page.getByRole("heading", { name: "Set password" })).toBeVisible();
	});

	test("shows mismatch validation for passphrase", async ({ page }) => {
		await page.goto("/dashboard/");
		await page.getByRole("link", { name: "Set passphrase" }).click();
		await page.getByRole("textbox", { name: "New passphrase", exact: true }).fill("passphrase-123");
		await page.getByRole("textbox", { name: "Confirm new passphrase" }).fill("passphrase-321");
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByText("New passphrase does not match confirmation.")).toBeVisible();
		await expect(page.getByRole("heading", { name: "Change passphrase" })).toBeVisible();
	});

	test("returns focus to passphrase trigger after closing dialog", async ({ page }) => {
		await page.goto("/dashboard/");
		const trigger = page.getByRole("link", { name: "Set passphrase" });
		await trigger.click();
		await page.getByRole("button", { name: "Close" }).click();

		await expect(trigger).toBeFocused();
	});

	test("set-password success removes no-password task", async ({ page }) => {
		test.skip(!allowStateMutation, "Enable E2E_ALLOW_STATE_MUTATION=1 to run state-mutation tests");

		await page.goto("/dashboard/");
		const setPasswordLink = page.getByRole("link", { name: "Set password now" });
		test.skip(!(await setPasswordLink.count()), "Set-password task only appears for temporary-password accounts");

		await setPasswordLink.click();
		await page.getByRole("textbox", { name: "New password" }).fill("ChangeMe1234");
		await page.getByRole("textbox", { name: "Confirm password" }).fill("ChangeMe1234");
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByRole("link", { name: "Set password now" })).toHaveCount(0);
	});

	test("passphrase success persists app status after reload", async ({ page }) => {
		test.skip(!allowStateMutation, "Enable E2E_ALLOW_STATE_MUTATION=1 to run state-mutation tests");

		await page.goto("/dashboard/");
		await page.getByRole("link", { name: "Set passphrase" }).click();
		await page.getByRole("textbox", { name: "New passphrase", exact: true }).fill("demo-passphrase");
		await page.getByRole("textbox", { name: "Confirm new passphrase" }).fill("demo-passphrase");
		await page.getByRole("button", { name: "Save" }).click();
		await page.reload();

		await expect(page.getByText(/protected by a passphrase/i)).toBeVisible();
	});
});
