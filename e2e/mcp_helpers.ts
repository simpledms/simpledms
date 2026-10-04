import { expect, type APIRequestContext, type Locator, type Page } from "@playwright/test";

import { fixturePath } from "./helpers";

export async function openNavigation(page: Page) {
	const menu = page.getByRole("button", { name: "Open main menu", exact: true }).first();
	if (await menu.getAttribute("aria-expanded") !== "true") await menu.click();
}

export async function prepareInbox(page: Page, spaceName: string, documentTypes: string[] = []) {
	await page.goto("/dashboard/");
	await openNavigation(page);
	const spaces = page.getByRole("link", { name: "Spaces", exact: true }).first();
	if (!(await spaces.isVisible())) await page.getByRole("button", { name: /^business / }).first().click();
	await spaces.click();
	await page.getByRole("link", { name: /Create Space/ }).first().click();
	await page.getByRole("textbox", { name: "Name", exact: true }).fill(spaceName);
	await page.getByRole("checkbox", { name: "Add me as space owner" }).check();
	for (const documentType of documentTypes) {
		await page.getByRole("checkbox", { name: documentType, exact: true }).check();
	}
	await page.getByRole("button", { name: "Create", exact: true }).click();
	await page.getByRole("heading", { name: spaceName, exact: true }).click();
	// wide screens open the Filters side sheet by default, which adds side_sheet to the query
	await expect(page).toHaveURL(/\/browse\/(\?.*)?$/);
	const inboxURL = page.url().replace(/\/browse\/(\?.*)?$/, "/inbox/");
	await page.goto(inboxURL);
	if ((page.viewportSize()?.width ?? 0) >= 768) await openNavigation(page);
	else {
		const menu = page.getByRole("button", { name: "Open main menu", exact: true }).first();
		if (await menu.getAttribute("aria-expanded") === "true") await menu.click();
	}
	await page.getByRole("link", { name: /Upload file/ }).first().click();
	const dialog = page.getByRole("dialog").filter({ hasText: "File upload" });
	await dialog.locator("input[type=file]").first().setInputFiles(fixturePath("upload-alpha.txt"));
	await expect(dialog.getByText("Upload complete")).toBeVisible();
	await dialog.getByRole("button", { name: /Cancel|Close|^close$/ }).last().click();
	await expect(page.getByRole("heading", { name: "upload-alpha.txt", exact: true })).toBeVisible();
	return inboxURL;
}

export async function createField(page: Page, inboxURL: string, name: string, type: string) {
	await page.goto(inboxURL.replace(/\/inbox\/$/, "/fields/"));
	await page.getByRole("link", { name: /Create field/ }).first().click();
	const dialog = page.getByRole("dialog").filter({ hasText: "Create field" });
	await dialog.getByRole("textbox", { name: "Name", exact: true }).fill(name);
	await dialog.getByRole("combobox", { name: "Type", exact: true }).selectOption({ label: type });
	await dialog.getByRole("button", { name: "Create", exact: true }).click();
	await expect(page.getByRole("heading", { name, exact: true })).toBeVisible();
}

export async function openMCPCredentials(page: Page) {
	await page.goto("/dashboard/");
	await openNavigation(page);
	await page.getByRole("link", { name: "MCP", exact: true }).first().click();
	await expect(page).toHaveURL(/\/mcp-credentials\/(\?.*)?$/);
}

export async function openCreateMCPCredential(page: Page) {
	await page.getByRole("button", { name: /Create MCP credential/ })
		.or(page.getByRole("link", { name: /Create MCP credential/ }))
		.filter({ visible: true }).first().click();
	return page.getByRole("dialog").filter({ hasText: "Create MCP credential" });
}

// Space radios are visually hidden inputs inside list item labels, so select via the label.
export async function selectSpace(form: Locator, spaceName: string) {
	const space = form.locator("label").filter({ hasText: spaceName });
	await space.click();
	await expect(space.locator("input[type=radio]")).toBeChecked();
}

export async function openCredentialAction(page: Page, label: string, action: string) {
	await page.locator("#mcpCredentials").getByRole("listitem").filter({ hasText: label })
		.getByRole("button", { name: "Actions" }).click();
	await page.getByRole("menu").filter({ visible: true }).getByRole("link", { name: action, exact: true }).click();
}

export async function rpc(
	request: APIRequestContext, token: string, method: string, params: object,
) {
	return request.post("/mcp", {
		headers: {
			Authorization: `Bearer ${token}`,
			Accept: "application/json, text/event-stream",
			"MCP-Protocol-Version": "2025-11-25",
		},
		data: { jsonrpc: "2.0", id: 1, method, params },
	});
}
