import { expect, test } from "@playwright/test";

import { createSpaceAndSelect, fixturePath, uniqueSuffix, uploadFileWithToolbar } from "./helpers";

test.describe("browse, upload, and filters", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("creates and opens a directory from browse", async ({ page }) => {
		const spaceName = `E2E Browse ${uniqueSuffix()}`;
		const dirName = `dir-${uniqueSuffix()}`;

		await createSpaceAndSelect(page, spaceName, ["Invoice", "Receipt"]);
		await page.getByRole("link", { name: "create_new_folder", exact: true }).click();
		await page.getByRole("textbox", { name: "Dir name" }).fill(dirName);
		await page.getByRole("button", { name: "Save" }).click();

		await expect(page.getByRole("heading", { name: dirName })).toBeVisible();
		await page.getByRole("link", { name: new RegExp(`folder ${dirName}`) }).click();
		await expect(page).toHaveURL(/\/browse\/[^/]+$/);
		await expect(page.getByRole("heading", { name: "No files or directories available yet." })).toBeVisible();
	});

	test("uploads a file and supports filename search", async ({ page }) => {
		const spaceName = `E2E Upload ${uniqueSuffix()}`;
		const fileName = `upload-alpha.txt`;

		await createSpaceAndSelect(page, spaceName, ["Invoice", "Receipt"]);
		await uploadFileWithToolbar(page, fixturePath(fileName));

		await expect(page.getByRole("heading", { name: fileName })).toBeVisible();
		const search = page.getByRole("searchbox", { name: "Search" });
		await search.fill(fileName);
		await expect(page.getByRole("heading", { name: fileName })).toBeVisible();

		await search.fill("no-match-search-token");
		await expect(page.getByRole("heading", { name: "No files or directories available yet." })).toBeVisible();
	});

	test("filters files by selected document type", async ({ page }) => {
		const spaceName = `E2E DocFilter ${uniqueSuffix()}`;
		const fileName = `upload-beta.txt`;

		await createSpaceAndSelect(page, spaceName, ["Invoice", "Receipt"]);
		await uploadFileWithToolbar(page, fixturePath(fileName));
		await page.getByRole("link", { name: new RegExp(`description ${fileName}`) }).click();
		await page.getByRole("button", { name: "description" }).click();
		await page.getByRole("link", { name: "Invoice" }).click();
		await expect(page.getByText("Document type selected.")).toBeVisible();
		await page.getByRole("button", { name: "close" }).click();
		await page.getByRole("link", { name: "close" }).click();

		await page.getByRole("link", { name: "category Document type" }).click();
		await page.getByRole("link", { name: "Receipt" }).click();
		await expect(page).toHaveURL(/document_type_id=/);
		await expect(page.getByRole("heading", { name: "No files or directories available yet." })).toBeVisible();

		await page.getByRole("link", { name: /Receipt close/ }).click();
		await page.getByRole("link", { name: "Invoice" }).click();
		await expect(page.getByRole("heading", { name: fileName })).toBeVisible();
	});
});
