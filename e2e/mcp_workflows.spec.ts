import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { readFile } from "node:fs/promises";

import { fixturePath, uniqueSuffix } from "./helpers";

async function openNavigation(page: Page) {
	const menu = page.getByRole("button", { name: "Open main menu", exact: true }).first();
	if (await menu.getAttribute("aria-expanded") !== "true") await menu.click();
}

async function prepareInbox(page: Page, spaceName: string) {
	await page.goto("/dashboard/");
	await openNavigation(page);
	const spaces = page.getByRole("link", { name: "Spaces", exact: true }).first();
	if (!(await spaces.isVisible())) {
		await page.getByRole("button", { name: /^business / }).first().click();
	}
	await spaces.click();
	await page.getByRole("link", { name: /Create space|^add$/ }).click();
	await page.getByRole("textbox", { name: "Name", exact: true }).fill(spaceName);
	await page.getByRole("checkbox", { name: "Add me as space owner" }).check();
	await page.getByRole("button", { name: "Save", exact: true }).click();
	await page.getByRole("heading", { name: spaceName, exact: true }).click();
	await expect(page).toHaveURL(/\/browse\/$/);
	const inboxURL = page.url().replace(/\/browse\/$/, "/inbox/");
	await page.goto(inboxURL);
	if ((page.viewportSize()?.width ?? 0) >= 768) {
		await openNavigation(page);
	} else {
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

async function rpc(
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

for (const device of [
	{ name: "desktop", viewport: { width: 1440, height: 1000 }, isMobile: false, hasTouch: false },
	{ name: "mobile", viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true },
]) {
	test.describe(`MCP ${device.name}`, () => {
		test.use({
			storageState: "e2e/.auth/admin.json",
			viewport: device.viewport,
			isMobile: device.isMobile,
			hasTouch: device.hasTouch,
		});

		test("@connect creates a scoped token, reads Inbox and revokes it", async ({ page }) => {
			const spaceName = `MCP ${uniqueSuffix()}`;
			const inboxURL = await prepareInbox(page, spaceName);
			const label = `Client ${uniqueSuffix()}`;
			await page.goto("/dashboard/account/");
			await page.getByRole("link", { name: /MCP credentials/ }).click();
			await expect(page).toHaveURL(/\/mcp-credentials\/$/);
			await page.getByRole("button", { name: /Create MCP credential/ }).click();
			const form = page.getByRole("dialog").filter({ hasText: "Create MCP credential" });
			await expect(form.getByRole("checkbox", { name: "Allow writes" })).not.toBeChecked();
			await form.getByRole("textbox", { name: "Label", exact: true }).fill(label);
			const destination = form.getByRole("combobox", { name: "Space", exact: true });
			const option = destination.locator("option").filter({ hasText: spaceName });
			await destination.selectOption((await option.getAttribute("value"))!);
			await destination.focus();
			await expect(destination).toBeFocused();
			await form.getByRole("button", { name: "Create", exact: true }).click();
			const secret = page.getByRole("dialog").filter({ hasText: "MCP credential created" });
			const tokenLink = secret.getByRole("link", { name: /^sdmcp_/ });
			const token = (await tokenLink.innerText()).trim();
			expect(token).toMatch(/^sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}$/);
			await tokenLink.focus();
			await expect(tokenLink).toBeFocused();
			await tokenLink.press("Enter");
			await expect(page.getByText(/Token copied to clipboard\.|Could not copy Token\./)).toBeVisible();
			const init = await rpc(page.request, token, "initialize", {
				protocolVersion: "2025-11-25", capabilities: {},
				clientInfo: { name: "simpledms-e2e", version: "1" },
			});
			expect(init.ok()).toBeTruthy();
			const list = await rpc(page.request, token, "tools/list", {});
			expect((await list.json()).result.tools.map((tool: { name: string }) => tool.name).sort())
				.toEqual(["get_file", "get_space", "list_inbox", "read_file_text", "upload_file"]);
			const response = await rpc(page.request, token, "tools/call", {
				name: "list_inbox", arguments: { limit: 10 },
			});
			const result = (await response.json()).result;
			expect(result.isError).not.toBe(true);
			expect(result.structuredContent.files.map((file: { name: string }) => file.name))
				.toContain("upload-alpha.txt");
			await page.reload();
			await expect(page.getByText(token, { exact: true })).toHaveCount(0);
			const credential = page.locator("#mcpCredentials").getByRole("listitem")
				.filter({ hasText: label });
			await credential.getByRole("button", { name: "Revoke", exact: true }).click();
			await expect(credential.getByText("Revoked", { exact: true })).toBeVisible();
			expect((await rpc(page.request, token, "tools/list", {})).status()).toBe(401);
			await page.goto(inboxURL);
			await expect(page.getByRole("heading", { name: "upload-alpha.txt", exact: true }))
				.toBeVisible();
		});

		test("@upload sends MCP bytes and opens them in Inbox", async ({ page }) => {
			const spaceName = `MCP upload ${uniqueSuffix()}`;
			const inboxURL = await prepareInbox(page, spaceName);
			await page.goto("/dashboard/account/");
			await page.getByRole("link", { name: /MCP credentials/ }).click();
			await page.getByRole("button", { name: /Create MCP credential/ }).click();
			const form = page.getByRole("dialog").filter({ hasText: "Create MCP credential" });
			await form.getByRole("textbox", { name: "Label", exact: true })
				.fill(`Uploader ${uniqueSuffix()}`);
			await form.getByRole("checkbox", { name: "Allow writes" }).check();
			const destination = form.getByRole("combobox", { name: "Space", exact: true });
			const option = destination.locator("option").filter({ hasText: spaceName });
			await destination.selectOption((await option.getAttribute("value"))!);
			await form.getByRole("button", { name: "Create", exact: true }).click();
			const secret = page.getByRole("dialog").filter({ hasText: "MCP credential created" });
			const token = (await secret.getByRole("link", { name: /^sdmcp_/ }).innerText()).trim();
			const init = await rpc(page.request, token, "initialize", {
				protocolVersion: "2025-11-25", capabilities: {},
				clientInfo: { name: "simpledms-e2e", version: "1" },
			});
			expect(init.ok()).toBeTruthy();

			const filename = `mcp-${uniqueSuffix()}.txt`;
			const content = "Uploaded through MCP\n";
			const response = await rpc(page.request, token, "tools/call", {
				name: "upload_file",
				arguments: {
					filename,
					content_base64: Buffer.from(content).toString("base64"),
				},
			});
			const result = (await response.json()).result;
			expect(result.isError).not.toBe(true);
			expect(result.structuredContent.filename).toBe(filename);
			await page.goto(inboxURL);
			await expect(page.getByRole("heading", { name: filename, exact: true })).toBeVisible();
			await expect(page.getByText("MCP", { exact: true }).first()).toBeVisible();
			await page.getByRole("button", { name: "Filter by source", exact: true }).click();
			const sourceFilter = page.getByRole("dialog").filter({ hasText: "Source" });
			await sourceFilter.getByRole("checkbox", { name: "MCP", exact: true }).check();
			await expect(page).toHaveURL(/source=MCP/);
			await expect(page.getByRole("heading", { name: filename, exact: true })).toBeVisible();
			await expect(page.getByRole("heading", { name: "upload-alpha.txt", exact: true }))
				.toHaveCount(0);

			await page.goto(result.structuredContent.url);
			await expect(page.getByRole("heading", { name: filename, exact: true })).toBeVisible();
			const downloadPromise = page.waitForEvent("download");
			await page.getByRole("link", { name: "Download", exact: true }).first().click();
			const download = await downloadPromise;
			expect((await readFile((await download.path())!)).toString()).toBe(content);
		});
	});
}
