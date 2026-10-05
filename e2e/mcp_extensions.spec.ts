import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";

import { openFileDetails, openFiltersTab, uniqueSuffix } from "./helpers";
import {
	openMCPCredentials, openCreateMCPCredential, prepareInbox, rpc, selectSpace,
} from "./mcp_helpers";

const devices = [
	{ name: "desktop", viewport: { width: 1440, height: 1000 }, isMobile: false, hasTouch: false },
	{ name: "mobile", viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true },
];

async function writableToken(page: Parameters<typeof prepareInbox>[0], spaceName: string) {
	await openMCPCredentials(page);
	const form = await openCreateMCPCredential(page);
	await form.getByRole("textbox", { name: "Client label", exact: true }).fill(`MCP ${uniqueSuffix()}`);
	await form.getByRole("switch", { name: "Allow writes" }).check();
	await selectSpace(form, spaceName);
	await form.getByRole("button", { name: "Create", exact: true }).click();
	const secret = page.getByRole("dialog").filter({ hasText: "MCP credential created" });
	const token = (await secret.getByRole("link", { name: /^sdmcp_/ }).innerText()).trim();
	const initialized = await rpc(page.request, token, "initialize", {
		protocolVersion: "2025-11-25", capabilities: {},
		clientInfo: { name: "simpledms-mcp-regression", version: "1" },
	});
	expect(initialized.ok()).toBeTruthy();
	return token;
}

async function tool(page: Parameters<typeof prepareInbox>[0], token: string, name: string, args: object) {
	const response = await rpc(page.request, token, "tools/call", { name, arguments: args });
	expect(response.ok()).toBeTruthy();
	const result = (await response.json()).result;
	expect(result.isError).not.toBe(true);
	expect(result.structuredContent).toBeTruthy();
	return result.structuredContent;
}

for (const device of devices) {
	test.describe(`MCP extensions ${device.name}`, () => {
		test.use({
			storageState: "e2e/.auth/admin.json",
			viewport: device.viewport,
			isMobile: device.isMobile,
			hasTouch: device.hasTouch,
		});
		test.setTimeout(90_000);

		test("keeps metadata definitions and typed field filters equal to MCP results", async ({ page }) => {
			const spaceName = `MCP parity ${uniqueSuffix()}`;
			const inboxURL = await prepareInbox(page, spaceName);
			const token = await writableToken(page, spaceName);
			const space = await tool(page, token, "get_space", {});
			const money = await tool(page, token, "create_property", {
				name: `MCP money ${uniqueSuffix()}`, type: "Money", unit: "CHF",
			});
			const date = await tool(page, token, "create_property", {
				name: `MCP date ${uniqueSuffix()}`, type: "Date",
			});
			const checkbox = await tool(page, token, "create_property", {
				name: `MCP checked ${uniqueSuffix()}`, type: "Checkbox",
			});
			const text = await tool(page, token, "create_property", {
				name: `MCP text ${uniqueSuffix()}`, type: "Text",
			});
			const tag = await tool(page, token, "create_tag", {
				name: `MCP tag ${uniqueSuffix()}`, type: "Simple",
			});
			const documentType = await tool(page, token, "create_document_type", {
				name: `MCP type ${uniqueSuffix()}`,
			});
			const renamed = `${money.name} edited`;
			await tool(page, token, "edit_property", { property_id: money.property_id, name: renamed });
			await tool(page, token, "edit_tag", { tag_id: tag.tag_id, name: `${tag.name} edited` });
			await tool(page, token, "rename_document_type", {
				document_type_id: documentType.document_type_id, name: `${documentType.name} edited`,
			});

			const uploaded = [];
			for (const [filename, amount, checked] of [
				[`mcp-money-${uniqueSuffix()}.txt`, 1250, true],
				[`mcp-checkbox-${uniqueSuffix()}.txt`, 425, false],
				[`mcp-missing-checkbox-${uniqueSuffix()}.txt`, 1250, undefined],
			] as const) {
				const file = await tool(page, token, "upload_file", {
					filename, content_base64: Buffer.from(`${filename}\n`).toString("base64"),
				});
				await tool(page, token, "file_inbox_document", {
					file_id: file.file_id, destination_directory_id: space.root_directory_id,
					filename,
				});
				await tool(page, token, "set_file_property", {
					file_id: file.file_id, property_id: money.property_id, money_minor_units: amount,
				});
				if (checked !== undefined) {
					await tool(page, token, "set_file_property", {
						file_id: file.file_id, property_id: checkbox.property_id, checkbox_value: checked,
					});
				}
				await tool(page, token, "set_file_property", {
					file_id: file.file_id, property_id: date.property_id, date_value: "2026-09-30",
				});
				uploaded.push({ file, checked, amount });
			}
			const moneyMatches = await tool(page, token, "search_files", {
				query: "", sort: "name", property_filters: [{
					property_id: money.property_id, operator: "equals", money_minor_units: 1250,
				}],
			});
			const uncheckedMatches = await tool(page, token, "search_files", {
				query: "", sort: "name", property_filters: [{
					property_id: checkbox.property_id, operator: "is_checked", checkbox_value: false,
				}],
			});
			expect(moneyMatches.files).toHaveLength(2);
			expect(uncheckedMatches.files).toHaveLength(2);
			const properties = await tool(page, token, "list_properties", {});
			expect(properties.properties.some((item: { name: string }) => item.name === renamed)).toBeTruthy();
			expect(properties.properties.some((item: { name: string }) => item.name === text.name)).toBeTruthy();

			await page.goto(inboxURL.replace(/\/inbox\/$/, "/fields/"));
			await expect(page.getByRole("heading", { name: renamed, exact: true })).toBeVisible();
			await expect(page.getByRole("heading", { name: text.name, exact: true })).toBeVisible();
			await page.goto(inboxURL.replace(/\/inbox\/$/, "/tags/"));
			await expect(page.getByText(`${tag.name} edited`, { exact: true })).toBeVisible();
			await page.goto(inboxURL.replace(/\/inbox\/$/, "/document-types/"));
			await expect(page.getByText(`${documentType.name} edited`, { exact: true })).toBeVisible();
			await tool(page, token, "delete_property", { property_id: text.property_id });
			await tool(page, token, "delete_tag", { tag_id: tag.tag_id });
			await tool(page, token, "delete_document_type", { document_type_id: documentType.document_type_id });
			await page.goto(inboxURL.replace(/\/inbox\/$/, "/fields/"));
			await expect(page.getByRole("heading", { name: text.name, exact: true })).toHaveCount(0);
			await page.goto(inboxURL.replace(/\/inbox\/$/, "/tags/"));
			await expect(page.getByText(`${tag.name} edited`, { exact: true })).toHaveCount(0);
			await page.goto(inboxURL.replace(/\/inbox\/$/, "/document-types/"));
			await expect(page.getByText(`${documentType.name} edited`, { exact: true })).toHaveCount(0);

			const rootURL = inboxURL.replace(/inbox\/$/, "browse/");
			await page.goto(rootURL);
			const filteredDialog = await openFiltersTab(page, "Fields");
			await filteredDialog.getByText(renamed, { exact: true }).click();
			const moneyInput = filteredDialog.getByRole("spinbutton", { name: renamed, exact: true });
			await expect(moneyInput).toBeVisible();
			await filteredDialog.getByText("Equals", { exact: true }).click();
			const moneyResponses = Promise.all([
				page.waitForResponse(response => response.url().includes("update-property-filter") && response.ok()),
				page.waitForResponse(response => response.url().includes("list-dir-partial") && response.ok()),
			]);
			await moneyInput.fill("12.50");
			await moneyResponses;
			const moneyNames = moneyMatches.files.map((file: { name: string }) => file.name);
			for (const name of moneyNames) await expect(page.locator("#fileList")).toContainText(name);
			await expect(page.locator("#fileList")).not.toContainText("mcp-checkbox-");

			// Start the Checkbox comparison from clean URL state. Numeric fields can emit a
			// second change request on blur after their debounced input save.
			await page.goto(rootURL);
			await openFiltersTab(page, "Fields");
			await filteredDialog.getByText(checkbox.name, { exact: true }).click();
			const uncheckedResponses = Promise.all([
				page.waitForResponse(response => response.url().includes("update-property-filter") &&
					new URLSearchParams(response.request().postData() ?? "").get("Value") === "false" && response.ok()),
				page.waitForResponse(response => response.url().includes("list-dir-partial") && response.ok()),
			]);
			await filteredDialog.getByText(`«${checkbox.name}» is not checked`, { exact: true }).click();
			await uncheckedResponses;
			const uncheckedNames = uncheckedMatches.files.map((file: { name: string }) => file.name);
			for (const name of uncheckedNames) await expect(page.locator("#fileList")).toContainText(name);
			await expect(page.locator("#fileList")).not.toContainText("mcp-money-");
		});

		test("round-trips download bytes, note history, and MCP filing through the browser", async ({ page }) => {
			const spaceName = `MCP organization ${uniqueSuffix()}`;
			const inboxURL = await prepareInbox(page, spaceName);
			const token = await writableToken(page, spaceName);
			const space = await tool(page, token, "get_space", {});
			const bytes = Buffer.from("MCP binary payload\u0000\u0001\u0002\n");
			const filename = `mcp-download-${uniqueSuffix()}.bin`;
			const uploaded = await tool(page, token, "upload_file", {
				filename, content_base64: bytes.toString("base64"),
			});
			const directoryName = `NewDirName ${uniqueSuffix()}`;
			const directory = await tool(page, token, "create_directory", {
				parent_directory_id: space.root_directory_id, name: directoryName,
			});
			await tool(page, token, "file_inbox_document", {
				file_id: uploaded.file_id, destination_directory_id: space.root_directory_id, filename,
			});
			const first = await tool(page, token, "create_document_note", {
				file_id: uploaded.file_id, title: "MCP review", body: "Original MCP note",
			});
			const replacement = await tool(page, token, "replace_document_note", {
				file_id: uploaded.file_id, note_id: first.note_id,
				title: "MCP review", body: "Replaced MCP note",
			});
			await tool(page, token, "edit_document_note", {
				file_id: uploaded.file_id, note_id: replacement.note_id,
				title: "MCP review", body: "Edited MCP note",
			});
			const deleted = await tool(page, token, "create_document_note", {
				file_id: uploaded.file_id, title: "Temporary MCP note", body: "Deleted MCP note",
			});
			await tool(page, token, "delete_document_note", {
				file_id: uploaded.file_id, note_id: deleted.note_id,
			});
			const history = await tool(page, token, "list_document_notes", {
				file_id: uploaded.file_id, show_history: true, limit: 20,
			});
			expect(history.notes).toHaveLength(3);
			await tool(page, token, "rename_file", {
				file_id: uploaded.file_id, new_filename: "renamed-by-mcp.bin",
			});
			await tool(page, token, "move_file", {
				file_id: uploaded.file_id, destination_directory_id: directory.directory_id,
				filename: "renamed-by-mcp.bin",
			});

			let downloaded = Buffer.alloc(0);
			let offset = 0;
			let versionNumber: number | undefined;
			for (;;) {
				const chunk = await tool(page, token, "download_file", {
					file_id: uploaded.file_id, offset, length: 4, version_number: versionNumber,
				});
				versionNumber = chunk.version_number;
				downloaded = Buffer.concat([downloaded, Buffer.from(chunk.content_base64, "base64")]);
				if (!chunk.has_more) break;
				offset = chunk.next_offset;
			}
			expect(downloaded.equals(bytes)).toBeTruthy();

			const found = await tool(page, token, "search_files", { query: "renamed-by-mcp", sort: "rank" });
			const filed = found.files.find((file: { file_id: string }) => file.file_id === uploaded.file_id);
			const location = await tool(page, token, "get_file", { file_id: uploaded.file_id });
			expect(location.parent_id).toBe(directory.directory_id);
			expect(filed).toBeTruthy();
			await page.goto(filed.url);
			await expect(page.getByRole("heading", { name: "renamed-by-mcp.bin", exact: true }).first()).toBeVisible();
			const details = await openFileDetails(page);
			await details.getByRole("tab", { name: "Notes", exact: true }).click();
			await expect(details.getByRole("button", { name: "Show deleted and replaced notes", exact: true }))
				.toBeVisible();
			await expect(details).toContainText("Edited MCP note");
			await details.getByRole("button", { name: "Show deleted and replaced notes", exact: true }).click();
			await expect(details).toContainText("Temporary MCP note");
			await expect(details.getByText("Deleted", { exact: true })).toBeVisible();
			await details.getByRole("button", { name: "Add note", exact: true }).click();
			const noteDialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Add note", exact: true }) });
			await noteDialog.getByRole("textbox", { name: "Title", exact: true }).fill("Browser correction");
			await noteDialog.getByRole("textbox", { name: "Note", exact: true }).fill("Browser-edited note");
			await noteDialog.getByRole("button", { name: "Save", exact: true }).click();
			await expect(details).toContainText("Browser-edited note");
			const current = await tool(page, token, "list_document_notes", {
				file_id: uploaded.file_id, show_history: false, limit: 20,
			});
			expect(current.notes.some((note: { body: string }) => note.body === "Browser-edited note")).toBeTruthy();
			await details.getByRole("tab", { name: "Info", exact: true }).click();
			await details.getByRole("button", { name: "close", exact: true }).click();
			const downloadPromise = page.waitForEvent("download");
			await page.getByRole("link", { name: "download", exact: true }).first().click();
			const browserDownload = await downloadPromise;
			expect((await readFile((await browserDownload.path())!)).equals(bytes)).toBeTruthy();
			await page.goto(inboxURL);
			await expect(page.getByRole("heading", { name: "renamed-by-mcp.bin", exact: true })).toHaveCount(0);
			await page.goto(filed.url);
			await expect(page.getByRole("heading", { name: "renamed-by-mcp.bin", exact: true }).first()).toBeVisible();
			// large screens open the details side sheet by default, which adds side_sheet to the query
			await expect(page).toHaveURL(
				new RegExp(`/browse/${directory.directory_id}/file/${uploaded.file_id}(\\?.*)?$`),
			);
			if (!device.isMobile) {
				await expect(page.getByText(directoryName, { exact: true }).last()).toBeVisible();
			}
		});
	});
}
