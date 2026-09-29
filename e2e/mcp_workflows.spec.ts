import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test";
import { readFile } from "node:fs/promises";

import { fixturePath, uniqueSuffix } from "./helpers";

async function openNavigation(page: Page) {
	const menu = page.getByRole("button", { name: "Open main menu", exact: true }).first();
	if (await menu.getAttribute("aria-expanded") !== "true") await menu.click();
}

async function prepareInbox(page: Page, spaceName: string, documentTypes: string[] = []) {
	await page.goto("/dashboard/");
	await openNavigation(page);
	const spaces = page.getByRole("link", { name: "Spaces", exact: true }).first();
	if (!(await spaces.isVisible())) {
		await page.getByRole("button", { name: /^business / }).first().click();
	}
	await spaces.click();
	await page.getByRole("link", { name: /Create space|^add$/ }).first().click();
	await page.getByRole("textbox", { name: "Name", exact: true }).fill(spaceName);
	await page.getByRole("checkbox", { name: "Add me as space owner" }).check();
	for (const documentType of documentTypes) {
		await page.getByRole("checkbox", { name: documentType, exact: true }).check();
	}
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

async function createField(page: Page, inboxURL: string, name: string, type: string) {
	await page.goto(inboxURL.replace(/\/inbox\/$/, "/fields/"));
	await page.getByRole("link", { name: /Add field|^add$/ }).first().click();
	const dialog = page.getByRole("dialog").filter({ hasText: "Add field" });
	await dialog.getByRole("textbox", { name: "Name", exact: true }).fill(name);
	await dialog.getByRole("combobox", { name: "Type", exact: true }).selectOption({ label: type });
	await dialog.getByRole("button", { name: "Save", exact: true }).click();
	await expect(page.getByRole("heading", { name, exact: true })).toBeVisible();
}

async function openMCPCredentials(page: Page) {
	await page.goto("/dashboard/");
	await openNavigation(page);
	await page.getByRole("link", { name: "MCP", exact: true }).first().click();
	await expect(page).toHaveURL(/\/mcp-credentials\/$/);
}

async function openCreateMCPCredential(page: Page) {
	await page.getByRole("button", { name: /Create MCP credential/ })
		.or(page.getByRole("link", { name: /Create MCP credential/ }))
		.filter({ visible: true }).first().click();
	return page.getByRole("dialog").filter({ hasText: "Create MCP credential" });
}

// Space radios are visually hidden inputs inside list item labels, so select via the label.
async function selectSpace(form: Locator, spaceName: string) {
	const space = form.locator("label").filter({ hasText: spaceName });
	await space.click();
	await expect(space.locator("input[type=radio]")).toBeChecked();
}

async function openCredentialAction(page: Page, label: string, action: string) {
	await page.locator("#mcpCredentials").getByRole("listitem").filter({ hasText: label })
		.getByRole("button", { name: "Actions" }).click();
	await page.getByRole("menu").filter({ visible: true })
		.getByRole("link", { name: action, exact: true }).click();
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
			await openMCPCredentials(page);
			const form = await openCreateMCPCredential(page);
			await expect(form.getByRole("switch", { name: "Allow writes" })).not.toBeChecked();
			await form.getByRole("textbox", { name: "Client label", exact: true }).fill(label);
			await selectSpace(form, spaceName);
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
				.toEqual([
					"assign_tag", "clear_document_type", "create_directory", "file_inbox_document",
					"get_document_type", "get_file", "get_space", "list_directory", "list_document_types",
					"list_inbox", "list_properties", "list_tags", "mark_inbox_file_done",
					"read_file_text", "remove_file_property", "search_files", "set_document_type",
					"set_file_property", "unassign_tag", "upload_file",
				]);
			const response = await rpc(page.request, token, "tools/call", {
				name: "list_inbox", arguments: { limit: 10 },
			});
			const result = (await response.json()).result;
			expect(result.isError).not.toBe(true);
			expect(result.structuredContent.files.map((file: { name: string }) => file.name))
				.toContain("upload-alpha.txt");
			await page.reload();
			await expect(page.getByText(token, { exact: true })).toHaveCount(0);
			// Credentials are grouped in one tab per Space.
			await page.getByRole("tab", { name: new RegExp(spaceName) }).click();
			const renamed = `${label} renamed`;
			await openCredentialAction(page, label, "edit Edit");
			const edit = page.getByRole("dialog").filter({ hasText: "Client label" });
			await edit.getByRole("textbox", { name: "Client label", exact: true }).fill(renamed);
			await edit.getByRole("button", { name: "Save", exact: true }).click();
			await expect(edit).toBeHidden();
			const credential = page.locator("#mcpCredentials").getByRole("listitem")
				.filter({ hasText: renamed });
			await expect(credential).toBeVisible();
			expect((await rpc(page.request, token, "tools/list", {})).ok()).toBeTruthy();
			page.once("dialog", (dialog) => dialog.accept());
			await openCredentialAction(page, renamed, "block Revoke");
			// Revoked credentials are hidden by the default active-only filter.
			await expect(credential).toHaveCount(0);
			await page.getByRole("button", { name: "Filter MCP credentials" }).click();
			const filter = page.getByRole("dialog").filter({ hasText: "Filter MCP credentials" });
			await filter.getByText("Revoked", { exact: true }).click();
			await expect(page).toHaveURL(/credential_status=revoked/);
			// The side sheet covers the tabs on mobile.
			await filter.getByRole("button", { name: "close", exact: true }).click();
			await page.getByRole("tab", { name: new RegExp(spaceName) }).click();
			await expect(credential.getByText(/Revoked:/)).toBeVisible();
			expect((await rpc(page.request, token, "tools/list", {})).status()).toBe(401);
			await page.goto(inboxURL);
			await expect(page.getByRole("heading", { name: "upload-alpha.txt", exact: true }))
				.toBeVisible();
		});

		test("@upload sends MCP bytes and opens them in Inbox", async ({ page }) => {
			const spaceName = `MCP upload ${uniqueSuffix()}`;
			const inboxURL = await prepareInbox(page, spaceName);
			await openMCPCredentials(page);
			const form = await openCreateMCPCredential(page);
			await form.getByRole("textbox", { name: "Client label", exact: true })
				.fill(`Uploader ${uniqueSuffix()}`);
			await form.getByRole("switch", { name: "Allow writes" }).check();
			await selectSpace(form, spaceName);
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

		test("@classify applies metadata and reflects browser corrections", async ({ page }) => {
			const spaceName = `MCP classify ${uniqueSuffix()}`;
			const inboxURL = await prepareInbox(page, spaceName, ["Invoice"]);
			const numberName = `MCP number ${uniqueSuffix()}`;
			const checkboxName = `MCP checkbox ${uniqueSuffix()}`;
			await createField(page, inboxURL, numberName, "Number");
			await createField(page, inboxURL, checkboxName, "Checkbox");
			await openMCPCredentials(page);
			const form = await openCreateMCPCredential(page);
			await form.getByRole("textbox", { name: "Client label", exact: true })
				.fill(`Classifier ${uniqueSuffix()}`);
			await form.getByRole("switch", { name: "Allow writes" }).check();
			await selectSpace(form, spaceName);
			await form.getByRole("button", { name: "Create", exact: true }).click();
			const secret = page.getByRole("dialog").filter({ hasText: "MCP credential created" });
			const token = (await secret.getByRole("link", { name: /^sdmcp_/ }).innerText()).trim();
			expect((await rpc(page.request, token, "initialize", {
				protocolVersion: "2025-11-25", capabilities: {},
				clientInfo: { name: "simpledms-e2e", version: "1" },
			})).ok()).toBeTruthy();

			const filename = `classified-${uniqueSuffix()}.txt`;
			const uploaded = (await (await rpc(page.request, token, "tools/call", {
				name: "upload_file",
				arguments: {
					filename,
					content_base64: Buffer.from("Classify through MCP\n").toString("base64"),
				},
			})).json()).result.structuredContent;
			const documentTypes = (await (await rpc(page.request, token, "tools/call", {
				name: "list_document_types", arguments: {},
			})).json()).result.structuredContent.document_types;
			const invoice = documentTypes.find((value: { name: string }) => value.name === "Invoice");
			const tags = (await (await rpc(page.request, token, "tools/call", {
				name: "list_tags", arguments: {},
			})).json()).result.structuredContent.tags;
			const open = tags.find((value: { name: string }) => value.name === "Open");
			const properties = (await (await rpc(page.request, token, "tools/call", {
				name: "list_properties", arguments: {},
			})).json()).result.structuredContent.properties;
			const invoiceNumber = properties.find(
				(value: { name: string }) => value.name === "Invoice number",
			);
			const invoiceDate = properties.find(
				(value: { name: string }) => value.name === "Invoice date",
			);
			const number = properties.find((value: { name: string }) => value.name === numberName);
			const checkbox = properties.find((value: { name: string }) => value.name === checkboxName);

			for (const call of [
				{ name: "set_document_type", arguments: {
					file_id: uploaded.file_id, document_type_id: invoice.document_type_id,
				} },
				{ name: "assign_tag", arguments: { file_id: uploaded.file_id, tag_id: open.tag_id } },
				{ name: "set_file_property", arguments: {
					file_id: uploaded.file_id, property_id: invoiceNumber.property_id, text_value: "MCP-1",
				} },
				{ name: "set_file_property", arguments: {
					file_id: uploaded.file_id, property_id: invoiceDate.property_id,
					date_value: "2026-09-29",
				} },
				{ name: "set_file_property", arguments: {
					file_id: uploaded.file_id, property_id: number.property_id, number_value: 0,
				} },
				{ name: "set_file_property", arguments: {
					file_id: uploaded.file_id, property_id: checkbox.property_id, checkbox_value: false,
				} },
			]) {
				const response = await rpc(page.request, token, "tools/call", call);
				expect((await response.json()).result.isError).not.toBe(true);
			}
			// Repeating desired-state writes must not toggle the type or duplicate the Tag.
			await rpc(page.request, token, "tools/call", {
				name: "set_document_type", arguments: {
					file_id: uploaded.file_id, document_type_id: invoice.document_type_id,
				},
			});
			await rpc(page.request, token, "tools/call", {
				name: "assign_tag", arguments: { file_id: uploaded.file_id, tag_id: open.tag_id },
			});

			await page.goto(uploaded.url);
			await page.getByRole("button", { name: "description" }).click();
			await expect(page.getByText("Invoice", { exact: true }).first()).toBeVisible();
			await expect(page.getByText("Open", { exact: true }).first()).toBeVisible();
			const reference = page.getByRole("textbox", { name: "Invoice number", exact: true });
			await expect(reference).toHaveValue("MCP-1");
			await reference.fill("MCP-2");
			await reference.press("Tab");
			const numberField = page.getByRole("spinbutton", { name: numberName, exact: true });
			await expect(numberField).toHaveValue("0");
			await numberField.fill("7");
			await numberField.press("Tab");
			const checkboxField = page.getByRole("checkbox", { name: checkboxName, exact: true });
			await expect(checkboxField).not.toBeChecked();
			await checkboxField.check();
			const date = page.getByLabel("Invoice date", { exact: true });
			await date.fill("");
			await date.press("Tab");
			await expect(page.getByText(/saved\.|removed\./).last()).toBeVisible();

			const file = (await (await rpc(page.request, token, "tools/call", {
				name: "get_file", arguments: { file_id: uploaded.file_id },
			})).json()).result.structuredContent;
			expect(file.document_type.document_type_id).toBe(invoice.document_type_id);
			expect(file.direct_tags.filter((value: { tag_id: string }) => value.tag_id === open.tag_id))
				.toHaveLength(1);
			expect(file.properties.find(
				(value: { property_id: string }) => value.property_id === invoiceNumber.property_id,
			).text_value).toBe("MCP-2");
			expect(file.properties.find(
				(value: { property_id: string }) => value.property_id === number.property_id,
			).number_value).toBe(7);
			expect(file.properties.find(
				(value: { property_id: string }) => value.property_id === checkbox.property_id,
			).checkbox_value).toBe(true);
			expect(file.properties.some(
				(value: { property_id: string }) => value.property_id === invoiceDate.property_id,
			)).toBe(false);
		});

		test("@file files an Inbox document and finds it again", async ({ page }) => {
			const spaceName = `MCP file ${uniqueSuffix()}`;
			const inboxURL = await prepareInbox(page, spaceName);
			await openMCPCredentials(page);
			const form = await openCreateMCPCredential(page);
			await form.getByRole("textbox", { name: "Client label", exact: true })
				.fill(`Filer ${uniqueSuffix()}`);
			await form.getByRole("switch", { name: "Allow writes" }).check();
			await selectSpace(form, spaceName);
			await form.getByRole("button", { name: "Create", exact: true }).click();
			const secret = page.getByRole("dialog").filter({ hasText: "MCP credential created" });
			const token = (await secret.getByRole("link", { name: /^sdmcp_/ }).innerText()).trim();
			expect((await rpc(page.request, token, "initialize", {
				protocolVersion: "2025-11-25", capabilities: {},
				clientInfo: { name: "simpledms-e2e", version: "1" },
			})).ok()).toBeTruthy();

			const space = (await (await rpc(page.request, token, "tools/call", {
				name: "get_space", arguments: {},
			})).json()).result.structuredContent;
			const inbox = (await (await rpc(page.request, token, "tools/call", {
				name: "list_inbox", arguments: {},
			})).json()).result.structuredContent;
			const document = inbox.files.find(
				(value: { name: string }) => value.name === "upload-alpha.txt",
			);
			const directory = (await (await rpc(page.request, token, "tools/call", {
				name: "create_directory",
				arguments: { parent_directory_id: space.root_directory_id, name: "Filed by MCP" },
			})).json()).result.structuredContent;
			const filed = (await (await rpc(page.request, token, "tools/call", {
				name: "file_inbox_document",
				arguments: {
					file_id: document.file_id,
					destination_directory_id: directory.directory_id,
					filename: "filed-by-mcp.txt",
				},
			})).json()).result.structuredContent;
			expect(filed.file_id).toBe(document.file_id);
			expect(filed.is_in_inbox).toBe(false);

			const remaining = (await (await rpc(page.request, token, "tools/call", {
				name: "list_inbox", arguments: {},
			})).json()).result.structuredContent.files;
			expect(remaining.some((value: { file_id: string }) => value.file_id === document.file_id))
				.toBe(false);
			const search = (await (await rpc(page.request, token, "tools/call", {
				name: "search_files", arguments: { query: "filed", sort: "rank" },
			})).json()).result.structuredContent;
			const found = search.files.find(
				(value: { file_id: string }) => value.file_id === document.file_id,
			);
			expect(found.name).toBe("filed-by-mcp.txt");
			await page.goto(found.url);
			await expect(page.getByRole("heading", { name: "filed-by-mcp.txt", exact: true }))
				.toBeVisible();
			await page.goto(inboxURL);
			await expect(page.getByRole("heading", { name: "filed-by-mcp.txt", exact: true }))
				.toHaveCount(0);
		});
	});
}
