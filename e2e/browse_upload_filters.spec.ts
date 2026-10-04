import { expect, test } from "@playwright/test";

import {
	createSpaceAndSelect,
	fixturePath,
	isLargeScreen,
	openFileDetails,
	openFiltersTab,
	uniqueSuffix,
	uploadFileWithToolbar,
} from "./helpers";

test.describe("browse, upload, and filters", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("creates and opens a directory from browse", async ({ page }) => {
		const spaceName = `E2E Browse ${uniqueSuffix()}`;
		const dirName = `dir-${uniqueSuffix()}`;

		await createSpaceAndSelect(page, spaceName, ["Invoice", "Receipt"]);
		await page.getByRole("link", { name: "create_new_folder Create folder", exact: true }).first().click();
		await page.getByRole("textbox", { name: "Folder name" }).fill(dirName);
		await page.getByRole("button", { name: "Create", exact: true }).click();

		await expect(page.getByRole("heading", { name: dirName })).toBeVisible();
		await page.getByRole("link", { name: new RegExp(`folder ${dirName}`) }).click();
		await expect(page).toHaveURL(/\/browse\/[^/]+$/);
		await expect(page.getByRole("heading", { name: "No files or folders available yet." })).toBeVisible();
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
		await expect(page.getByRole("heading", { name: "No files or folders available yet." })).toBeVisible();
	});

	test("highlights the inbox filter button while a source filter is active", async ({ page }) => {
		await createSpaceAndSelect(page, `E2E InboxFilter ${uniqueSuffix()}`);
		await page.goto(page.url().replace(/\/browse\/(\?.*)?$/, "/inbox/"));

		const filterButtonIcon = page.getByRole("button", { name: "Filter by source" }).locator("i");
		await expect(filterButtonIcon).not.toHaveClass(/\bfill\b/);

		const sourceFilter = page.locator("#filterSourceDialog");
		if (!isLargeScreen(page)) {
			await page.getByRole("button", { name: "Filter by source" }).click();
		}
		await sourceFilter.getByText("Web upload", { exact: true }).click();
		await expect(page).toHaveURL(/source=/);
		await expect(filterButtonIcon).toHaveClass(/\bfill\b/);

		await sourceFilter.getByText("Web upload", { exact: true }).click();
		await expect(page).not.toHaveURL(/source=/);
		await expect(filterButtonIcon).not.toHaveClass(/\bfill\b/);
	});

	test("filters files by selected document type", async ({ page }) => {
		const spaceName = `E2E DocFilter ${uniqueSuffix()}`;
		const fileName = `upload-beta.txt`;

		await createSpaceAndSelect(page, spaceName, ["Invoice", "Receipt"]);
		await uploadFileWithToolbar(page, fixturePath(fileName));
		await page.getByRole("link", { name: new RegExp(`description ${fileName}`) }).click();
		const details = await openFileDetails(page);
		await details.getByRole("link", { name: "Invoice" }).click();
		await expect(page.getByText("Document type selected.")).toBeVisible();
		await page.getByRole("button", { name: "close" }).click();
		await page.getByRole("link", { name: "close" }).click();

		const filtersButtonIcon = page.locator("#filtersBtn i");
		await expect(filtersButtonIcon).not.toHaveClass(/\bfill\b/);

		const filters = await openFiltersTab(page, "Document type");
		await filters.getByRole("link", { name: "Receipt" }).click();
		await expect(page).toHaveURL(/document_type_id=/);
		await expect(page.getByRole("heading", { name: "No files or folders available yet." })).toBeVisible();
		await expect(filtersButtonIcon).toHaveClass(/\bfill\b/);
		const documentTypeBadge = filters.locator("#filterTabBadge-document-type");
		await expect(documentTypeBadge).toBeVisible();

		await filters.getByRole("link", { name: /Receipt close/ }).click();
		await filters.getByRole("link", { name: "Invoice" }).click();
		await expect(page.getByRole("heading", { name: fileName })).toBeVisible();

		await filters.getByRole("button", { name: /Reset/ }).click();
		await expect(page.getByText("Filters reset.")).toBeVisible();
		await expect(page).not.toHaveURL(/document_type_id=/);
		await expect(filtersButtonIcon).not.toHaveClass(/\bfill\b/);
		await expect(documentTypeBadge).toBeHidden();
		await expect(filters.getByRole("link", { name: "Receipt" })).toBeVisible();
		await expect(page.getByRole("heading", { name: fileName })).toBeVisible();
	});
});
