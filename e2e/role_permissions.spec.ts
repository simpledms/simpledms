import { expect, test } from "@playwright/test";

import { createSpaceAndSelect, openSpaceMenu, uniqueSuffix } from "./helpers";

test.describe("role permissions (owner)", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("owner sees and can open space admin sections", async ({ page }) => {
		const spaceName = `E2E Owner ${uniqueSuffix()}`;
		await createSpaceAndSelect(page, spaceName, ["Invoice", "Receipt"]);
		const base = page.url().match(/(\/org\/[^/]+\/space\/[^/]+)/)?.[1];
		if (!base) {
			throw new Error("Failed to detect space base URL");
		}

		await openSpaceMenu(page);
		await expect(page.getByRole("link", { name: "category Document types", exact: true })).toBeVisible();
		await expect(page.getByRole("link", { name: "label Tags", exact: true })).toBeVisible();
		await expect(page.getByRole("link", { name: "tune Fields", exact: true })).toBeVisible();
		await expect(page.getByRole("link", { name: "person Users", exact: true })).toBeVisible();
		await expect(page.getByRole("link", { name: "delete Trash", exact: true })).toBeVisible();

		await page.goto(`${base}/document-types/`);
		await expect(page).toHaveURL(/\/space\/[^/]+\/document-types\/$/);
		await expect(page.getByRole("heading", { name: "Document types" })).toBeVisible();

		await page.goto(`${base}/tags/`);
		await expect(page).toHaveURL(/\/space\/[^/]+\/tags\/$/);
		await expect(page.getByRole("heading", { name: "Tags" })).toBeVisible();

		await page.goto(`${base}/manage-users/`);
		await expect(page).toHaveURL(/\/space\/[^/]+\/manage-users\/$/);
		await expect(page.getByRole("heading", { name: /Users «/ })).toBeVisible();

		await page.goto(`${base}/trash/`);
		await expect(page).toHaveURL(/\/space\/[^/]+\/trash\/$/);
		await expect(page.getByRole("heading", { name: /Trash/ })).toBeVisible();
	});
});
