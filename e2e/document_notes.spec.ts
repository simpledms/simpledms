import { expect, test, type Locator, type Page } from "@playwright/test";

import {
	fixturePath,
	loginEmail,
	uniqueSuffix,
} from "./helpers";

const historyLabel = "Show deleted and replaced notes";
const fileName = "upload-alpha.txt";
const entries = (page: Page) => page.locator('#documentNotes [id^="documentNote-"]');
const history = (page: Page) => page.getByRole("button", { name: historyLabel, exact: true });
const note = (page: Page, text: string) => entries(page).filter({ hasText: text });

async function showHistory(page: Page) {
	if (await history(page).getAttribute("aria-pressed") !== "true") {
		await history(page).click();
	}
	await expect(history(page)).toHaveAttribute("aria-pressed", "true");
}

async function expandNavigation(page: Page) {
	const toggle = page.getByRole("button", { name: "Open main menu", exact: true }).first();
	if (await toggle.getAttribute("aria-expanded") !== "true") await toggle.click();
}

async function createSpaceAndSelect(page: Page, name: string) {
	await page.goto("/dashboard/");
	await expandNavigation(page);
	const organization = page.getByRole("button", { name: /^business / });
	if (!(await page.getByRole("link", { name: "Spaces", exact: true }).isVisible())) {
		await organization.click();
	}
	await page.getByRole("link", { name: "Spaces", exact: true }).click();
	await page.getByRole("link", { name: /Create space|^add$/ }).click();
	await page.getByRole("textbox", { name: "Name", exact: true }).fill(name);
	const owner = page.getByRole("checkbox", { name: "Add me as space owner" });
	expect(await owner.evaluate(input =>
		input.parentElement!.parentElement!.parentElement!.getBoundingClientRect().height,
	)).toBe(48);
	await owner.check();
	await page.getByRole("button", { name: "Save", exact: true }).click();
	await page.getByRole("heading", { name, exact: true }).click();
	await expect(page).toHaveURL(/\/browse\/$/);
}

async function uploadFileWithToolbar(page: Page, file: string) {
	if ((page.viewportSize()?.width ?? 0) < 768) {
		const toggle = page.getByRole("button", { name: "Open main menu", exact: true }).first();
		if (await toggle.getAttribute("aria-expanded") === "true") await toggle.click();
	} else {
		await expandNavigation(page);
	}
	await page.getByRole("link", { name: /Upload file/ }).first().click();
	const dialog = page.getByRole("dialog").filter({ hasText: "File upload" });
	await dialog.locator("input[type=file]").first().setInputFiles(file);
	await expect(dialog.getByText("Upload complete")).toBeVisible();
	await cancelDialog(dialog);
}

async function selectNotes(page: Page) {
	// Details is open automatically only on extra-large screens.
	if (!(await page.getByRole("tab", { name: "Notes", exact: true }).isVisible())) {
		await page.getByRole("button", { name: /Show details|^description$/ }).click();
	}
	await page.getByRole("tab", { name: "Notes", exact: true }).click();
	await expect(page.locator("#documentNotes")).toBeVisible();
	await expect(page.getByRole("tab", { name: "Notes", exact: true }))
		.toHaveAttribute("aria-selected", "true");
}

async function openDocument(page: Page, name = fileName) {
	await page.getByRole("heading", { name, exact: true }).click();
	await selectNotes(page);
}

async function setupDocument(page: Page, inbox = false) {
	// Read the authenticated user's actual tenant display name through the existing UI.
	await page.goto("/dashboard/");
	await expandNavigation(page);
	if (!(await page.getByRole("link", { name: "Users", exact: true }).isVisible())) {
		await page.getByRole("button", { name: /^business / }).click();
	}
	await page.getByRole("link", { name: /Users/ }).click();
	const user = page.locator("#userListPartial").getByRole("listitem")
		.filter({ hasText: loginEmail });
	const author = (await user.getByRole("heading").innerText()).trim();
	expect(author).not.toBe("");
	await createSpaceAndSelect(page, `E2E Notes ${uniqueSuffix()}`);
	const browseURL = page.url();
	if (inbox) await navigateSection(page, "Inbox");
	const listURL = page.url();
	await uploadFileWithToolbar(page, fixturePath(fileName));
	await openDocument(page);
	await expect(entries(page)).toHaveCount(0);
	await expect(history(page)).toHaveAttribute("aria-pressed", "false");
	await expect(history(page)).toHaveText("history");
	await expect(page.getByRole("button", { name: "Add note", exact: true })).toHaveText("add");
	await expect(page.locator("#documentNotes").getByRole("switch")).toHaveCount(0);
	return { author, browseURL, listURL };
}

async function navigateSection(page: Page, section: "Inbox" | "Trash") {
	await expandNavigation(page);
	await page.getByRole("link", { name: new RegExp(`\\b${section}\\b`) }).click();
	await expect(page).toHaveURL(new RegExp(`/${section.toLowerCase()}/`));
}

async function openNoteDialog(page: Page, action: string, entry?: Locator) {
	if (entry) {
		await expect(entry).toHaveClass(/js-list-item/);
		await expect(entry.getByRole("menu")).not.toBeVisible();
		if ((page.viewportSize()?.width ?? 0) < 768) {
			await entry.getByRole("button", { name: "Actions", exact: true }).click();
		} else {
			await entry.click({ button: "right" });
		}
		await expect(entry.getByRole("menu")).toBeVisible();
	}
	if (entry) {
		await entry.getByRole("link", { name: action, exact: true }).click();
	} else {
		await page.getByRole("button", { name: action, exact: true }).click();
	}
	const heading = action === "Add note" ? action : `${action} note`;
	const dialog = page.getByRole("dialog").filter({
		has: page.getByRole("heading", { name: heading, exact: true }),
	});
	await expect(dialog).toBeVisible();
	return dialog;
}

async function saveNote(
	page: Page, text: string, action = "Add note", entry?: Locator,
	title = "Document review",
) {
	const oldTitle = entry && await entry.getByRole("button").first().locator("div").nth(0).innerText();
	const oldBody = entry && await entry.getByRole("button").first().locator("div").nth(1).innerText();
	const dialog = await openNoteDialog(page, action, entry);
	const titleInput = dialog.getByRole("textbox", { name: "Title", exact: true });
	await expect(titleInput).toBeFocused();
	if (entry) {
		await expect(titleInput).toHaveValue(oldTitle!);
		await expect(dialog.getByRole("textbox", { name: "Note", exact: true })).toHaveValue(oldBody!);
	}
	await titleInput.fill(title);
	await dialog.getByRole("textbox", { name: "Note", exact: true }).fill(text);
	await dialog.getByRole("button", {
		name: action === "Replace" ? "Replace" : "Save", exact: true,
	}).click();
	await expect(dialog).not.toBeVisible();
	await expect(note(page, text)).toHaveCount(1);
	await expect(note(page, text)).toBeVisible();
	await expect(note(page, text).getByRole("button").first().locator("div").nth(0)).toHaveText(title);
	await expect(page.getByRole("tab", { name: "Notes", exact: true }))
		.toHaveAttribute("aria-selected", "true");
	const id = await note(page, text).getAttribute("id");
	expect(id).toMatch(/^documentNote-.+/);
	return page.locator(`[id="${id}"]`);
}

async function cancelDialog(dialog: Locator) {
	await dialog.getByRole("button", { name: /^(close|Cancel)$/i }).first().click();
	await expect(dialog).not.toBeVisible();
}

async function deleteNote(page: Page, entry: Locator) {
	const dialog = await openNoteDialog(page, "Delete", entry);
	await dialog.getByRole("button", { name: "Delete", exact: true }).click();
	await expect(dialog).not.toBeVisible();
}

async function openNoteDetails(page: Page, entry: Locator) {
	await entry.getByRole("button").first().click();
	const dialog = page.locator("#documentNoteDetailsDialog");
	await expect(dialog).toBeVisible();
	return dialog;
}

async function expectReplacement(
	page: Page, entry: Locator, successor: Locator, oldBody: string, successorBody: string,
) {
	const successorTitle = await successor.getByRole("button").first().locator("div").nth(0).innerText();
	await expect(entry).not.toContainText(oldBody);
	await expect(entry.locator("em")).toHaveText(`Replaced by: ${successorTitle}`);
	await expect(entry.locator("em")).toHaveCSS("font-style", "italic");
	const dialog = await openNoteDetails(page, entry);
	await expect(dialog).toContainText(oldBody);
	await expect(dialog.getByRole("link", { name: "View replacement note" }))
		.toHaveAttribute("href", `#${await successor.getAttribute("id")}`);
	await dialog.getByRole("link", { name: "View replacement note" }).click();
	await expect(dialog).toContainText(successorBody);
	await expect(dialog).toContainText(successorTitle);
	await expect(dialog.getByRole("heading", { name: successorTitle, exact: true })).toBeVisible();
	await expect(dialog).toHaveJSProperty("open", true);
	expect(await dialog.evaluate(element => element.matches(":modal"))).toBe(true);
	await cancelDialog(dialog);
}

async function expectAuthor(page: Page, entry: Locator, author: string) {
	await expect(entry).not.toContainText(author);
	const dialog = await openNoteDetails(page, entry);
	await expect(dialog).toContainText(`Author: ${author}`);
	await cancelDialog(dialog);
}

async function expectReadOnly(entry: Locator) {
	await expect(entry.getByRole("button", { name: "Actions", exact: true })).toHaveCount(0);
	await expect(entry.getByRole("menu")).toHaveCount(0);
	for (const action of ["Edit", "Replace", "Delete"]) {
		await expect(entry.getByRole("link", { name: action, exact: true })).toHaveCount(0);
	}
}

async function expectNoOverflow(page: Page) {
	await expect.poll(() => page.evaluate(() =>
		document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1,
	)).toBe(true);
	await expect.poll(() => page.locator("#documentNotes").evaluate(element =>
		element.scrollWidth <= element.clientWidth + 1,
	)).toBe(true);
}

for (const device of [
	{ name: "desktop", viewport: { width: 1440, height: 1000 }, isMobile: false, hasTouch: false },
	{ name: "mobile", viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true },
]) {
	test.describe(`document notes - ${device.name}`, () => {
		test.use({
			storageState: "e2e/.auth/admin.json",
			viewport: device.viewport,
			isMobile: device.isMobile,
			hasTouch: device.hasTouch,
		});
		test.setTimeout(90_000);

		test("creates a note from the Fields-style empty state", async ({ page }) => {
			const { author } = await setupDocument(page);
			const emptyHeading = page.getByRole("heading", { name: "No notes available.", exact: true });
			await expect(emptyHeading).toBeVisible();
			await expectNoOverflow(page);
			await page.getByRole("button", { name: "add Add note", exact: true }).click();
			const dialog = page.getByRole("dialog").filter({
				has: page.getByRole("heading", { name: "Add note", exact: true }),
			});
			await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toBeFocused();
			await dialog.getByRole("textbox", { name: "Title", exact: true }).fill("Review decision");
			await dialog.getByRole("textbox", { name: "Note", exact: true }).fill("First note");
			await expect(dialog.locator("label:has(#documentNoteBody) > span")).toBeVisible();
			await dialog.getByRole("button", { name: "Save", exact: true }).click();
			await expect(emptyHeading).toHaveCount(0);
			await expect(note(page, "First note")).toBeVisible();
			const summary = note(page, "First note").getByRole("button").first().locator("div");
			await expect(summary.nth(0)).toHaveText("Review decision");
			await expect(summary.nth(1)).toHaveText("First note");
			await expect(note(page, "First note")).not.toContainText(author);
			const details = await openNoteDetails(page, note(page, "First note"));
			await expect(details).toContainText("Review decision");
			await expect(details).toContainText(`Author: ${author}`);
			await expect(details).toContainText("Created:");
			await cancelDialog(details);
			await deleteNote(page, note(page, "First note"));
			await expect(emptyHeading).toBeVisible();
			await showHistory(page);
			await expect(emptyHeading).toHaveCount(0);
		});

		test("matches textfield label layout in empty, focused, and populated states", async ({ page }) => {
			await setupDocument(page);
			const dialog = await openNoteDialog(page, "Add note");
			const title = dialog.getByRole("textbox", { name: "Title", exact: true });
			const body = dialog.getByRole("textbox", { name: "Note", exact: true });
			const layout = (control: Locator) => control.evaluate(element => {
				const label = element.closest("label")!;
				const caption = label.querySelector("span")!;
				const container = label.parentElement!;
				const box = container.getBoundingClientRect();
				const inputBox = element.getBoundingClientRect();
				const captionBox = caption.getBoundingClientRect();
				const style = getComputedStyle(caption);
				const outline = getComputedStyle(container);
				return {
					contentX: inputBox.x - box.x,
					contentY: inputBox.y - box.y,
					labelX: captionBox.x - box.x,
					labelY: captionBox.y - box.y,
					fontSize: style.fontSize,
					lineHeight: style.lineHeight,
					color: style.color,
					background: style.backgroundColor,
					outlineStyle: outline.outlineStyle,
					outlineWidth: outline.outlineWidth,
					outlineColor: outline.outlineColor,
					borderRadius: outline.borderRadius,
					innerOutline: getComputedStyle(element).outlineStyle,
				};
			});
			for (const colorScheme of ["light", "dark"] as const) {
				await page.emulateMedia({ colorScheme });
				for (const state of ["empty", "focused", "populated"] as const) {
					for (const control of [title, body]) {
						await control.fill(state === "populated" ? "Review notes" : "");
						await control.blur();
					}
					if (state === "focused") await title.focus();
					const caption = title.locator("..").locator("span");
					await expect(caption).toHaveCSS("top", state === "empty" ? "0px" : "-24px");
					await caption.evaluate(element => Promise.all(
						element.getAnimations().map(animation => animation.finished),
					));
					const expected = await layout(title);
					if (state === "focused") {
						// The textarea must use the same single outline on keyboard focus too.
						await page.keyboard.press("Tab");
						await expect(body).toBeFocused();
					}
					await expect.poll(() => layout(body)).toEqual(expected);
					await expectNoOverflow(page);
				}
			}
			await body.fill(Array(40).fill("Long multiline note").join("\n"));
			await body.press("Control+End");
			await expect(body.locator("..").locator("span")).toBeVisible();
			await expectNoOverflow(page);
			await cancelDialog(dialog);
		});

		test("places add before history in the Notes toolbar", async ({ page }) => {
			await setupDocument(page);
			const addBox = await page.getByRole("button", { name: "Add note", exact: true })
				.boundingBox();
			const historyBox = await history(page).boundingBox();
			expect(addBox).not.toBeNull();
			expect(historyBox).not.toBeNull();
			expect(addBox!.x).toBeLessThan(historyBox!.x);
		});

		if (device.name === "desktop") {
			test("uses Material 3 standard icon colors when selected and hovered", async ({ page }) => {
				await setupDocument(page);
				for (const colorScheme of ["light", "dark"] as const) {
					await page.emulateMedia({ colorScheme });
					const colors = await history(page).evaluate(button => {
						const style = getComputedStyle(button);
						return {
							primary: style.getPropertyValue("--md-sys-color-primary")
								.trim().split(/\s+/).join(", "),
							neutral: style.getPropertyValue("--md-sys-color-on-surface-variant")
								.trim().split(/\s+/).join(", "),
						};
					});
					for (const selected of [true, false]) {
						await history(page).click();
						await expect(history(page)).toHaveAttribute("aria-pressed", String(selected));
						await page.mouse.move(0, 0);
						await history(page).evaluate(button => button.blur());
						const color = selected ? colors.primary : colors.neutral;
						await expect(history(page).locator("i")).toHaveCSS("color", `rgb(${color})`);
						// Standard icon buttons have no persistent filled container.
						await expect(history(page).locator(":scope > div"))
							.toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
						await history(page).hover();
						await expect(history(page).locator("i")).toHaveCSS("color", `rgb(${color})`);
						await expect(history(page).locator(":scope > div"))
							.toHaveCSS("background-color", `rgba(${color}, 0.08)`);
					}
					await expect(page.getByRole("button", { name: "Add note", exact: true }).locator("i"))
						.toHaveCSS("color", `rgb(${colors.neutral})`);
				}
			});
		}

		test("creates plain-text notes, edits in place, and persists after reopening", async ({ page }) => {
			const { author, browseURL } = await setupDocument(page);
			const firstText = 'First line\n<strong>Literal HTML, not markup</strong>\n' + "long".repeat(80);
			const first = await saveNote(page, firstText, "Add note", undefined, "<strong>Review</strong>");
			const firstID = await first.getAttribute("id");
			await expectAuthor(page, first, author);
			expect(await first.textContent()).toContain(firstText);
			await expect(first.locator("strong")).toHaveCount(0);
			await expectNoOverflow(page);
			const fullText = await openNoteDetails(page, first);
			await expect(fullText).toContainText(firstText);
			await expect(fullText.locator("strong")).toHaveCount(0);
			expect(await fullText.locator("#documentNoteDetailsContent").evaluate(element =>
				element.scrollWidth <= element.clientWidth + 1,
			)).toBe(true);
			await cancelDialog(fullText);
			const second = await saveNote(page, "Second note\nNewest first");
			const secondID = await second.getAttribute("id");
			await expect(entries(page)).toHaveCount(2);
			await expect(entries(page).first()).toHaveAttribute("id", secondID!);
			await saveNote(page, "Corrected first note\nStill the original entry", "Edit", first, "Decision");
			await expectAuthor(page, first, author);
			await expect(first.locator('[title^="Created:"]'))
				.toHaveAttribute("title", /^Created: .+\nAuthor: /);
			expect(await first.locator('[title^="Created:"]').getAttribute("title"))
				.toContain(author);
			await expect(first).not.toContainText("Edited by");
			const details = await openNoteDetails(page, first);
			await expect(details).toContainText("Author:");
			await expect(details).toContainText("Created:");
			await expect(details).toContainText("Edited by");
			await expect(details).toContainText("Corrected first note");
			await cancelDialog(details);
			await expect(entries(page).last()).toHaveAttribute("id", firstID!);
			await expect(entries(page)).toHaveCount(2);
			await expectNoOverflow(page);
			await page.reload();
			await selectNotes(page);
			await expect(first).toContainText("Corrected first note");
			await expectAuthor(page, second, author);
			await expect(first).toContainText("Decision");
			await page.getByRole("tab", { name: "Info", exact: true }).click();
			await selectNotes(page);
			await expect(entries(page)).toHaveCount(2);
			await page.goto(browseURL);
			await openDocument(page);
			await expect(entries(page).first()).toHaveAttribute("id", secondID!);
			await expectAuthor(page, first, author);
		});

		test("cancel and invalid content leave create, edit, replace, and delete unchanged", async ({ page }) => {
			await setupDocument(page);
			const original = await saveNote(page, "Original remains current");
			for (const action of ["Add note", "Edit", "Replace", "Delete"]) {
				const dialog = await openNoteDialog(
					page, action, action === "Add note" ? undefined : original,
				);
				if (action !== "Delete") {
					await dialog.getByRole("textbox", { name: "Note", exact: true })
						.fill("Cancelled draft");
				}
				await cancelDialog(dialog);
				await expect(original).toContainText("Original remains current");
				await expect(entries(page)).toHaveCount(1);
			}
			for (const action of ["Add note", "Edit", "Replace"]) {
				for (const invalid of ["", " \n\t "]) {
					const dialog = await openNoteDialog(
						page, action, action === "Add note" ? undefined : original,
					);
					const input = dialog.getByRole("textbox", { name: "Note", exact: true });
					await dialog.getByRole("textbox", { name: "Title", exact: true }).fill("Valid title");
					await input.fill(invalid);
					await dialog.getByRole("button", {
						name: action === "Replace" ? "Replace" : "Save", exact: true,
					}).click();
					await expect(dialog).toBeVisible();
					await expect(page.getByText(/note.*(required|empty|whitespace)|enter.*note/i)
						.first()).toBeVisible();
					await cancelDialog(dialog);
					await expect(entries(page)).toHaveCount(1);
					await expect(original).toContainText("Original remains current");
				}
				const dialog = await openNoteDialog(
					page, action, action === "Add note" ? undefined : original,
				);
				await dialog.getByRole("textbox", { name: "Title", exact: true }).fill("   ");
				await dialog.getByRole("textbox", { name: "Note", exact: true }).fill("Valid body");
				await dialog.getByRole("button", {
					name: action === "Replace" ? "Replace" : "Save", exact: true,
				}).click();
				await expect(dialog).toBeVisible();
				await expect(page.getByText("Note title must not be empty.", { exact: true })).toBeVisible();
				await cancelDialog(dialog);
				await expect(original).toContainText("Document review");
				await expect(original).toContainText("Original remains current");
			}
			await showHistory(page);
			await expect(entries(page)).toHaveCount(1);
		});

		test("opens note actions and edits with the keyboard", async ({ page }) => {
			await setupDocument(page);
			const entry = await saveNote(page, "Keyboard menu note");
			await entry.getByRole("button").first().focus();
			await page.keyboard.press("Enter");
			const details = page.locator("#documentNoteDetailsDialog");
			await expect(details).toBeVisible();
			await expect(details).toContainText("Keyboard menu note");
			await cancelDialog(details);
			const trigger = entry.getByRole("button", { name: "Actions", exact: true });
			await trigger.focus();
			await page.keyboard.press("Enter");
			await expect(entry.getByRole("menu")).toBeVisible();
			await expect(details).toHaveCount(0);
			await page.keyboard.press("Escape");
			await expect(entry.getByRole("menu")).not.toBeVisible();
			await expect(trigger).toBeFocused();
			await page.keyboard.press("Enter");
			await expect(entry.getByRole("menu")).toBeVisible();
			await entry.getByRole("link", { name: "Edit", exact: true }).focus();
			await page.keyboard.press("Enter");
			const dialog = page.getByRole("dialog").filter({
				has: page.getByRole("heading", { name: "Edit note", exact: true }),
			});
			await expect(dialog).toBeVisible();
			await dialog.getByRole("textbox", { name: "Note", exact: true })
				.fill("Keyboard updated note");
			await dialog.getByRole("button", { name: "Save", exact: true }).focus();
			await page.keyboard.press("Enter");
			await expect(dialog).not.toBeVisible();
			await expect(entry).toContainText("Keyboard updated note");
			await expect(entry.getByRole("menu")).not.toBeVisible();
		});

		test("keeps mixed replacement history and keyboard toggle state through commands", async ({ page }) => {
			const { author } = await setupDocument(page);
			const original = await saveNote(page, "Original predecessor", "Add note", undefined, "Draft");
			const successor = await saveNote(page, "First successor", "Replace", original, "Revised draft");
			await expect(original).toHaveCount(0);
			const current = await saveNote(page, "Second successor", "Replace", successor, "Final decision");
			await expect(successor).toHaveCount(0);
			await history(page).focus();
			await page.keyboard.press("Space");
			await expect(history(page)).toHaveAttribute("aria-pressed", "true");
			await expect(entries(page)).toHaveCount(3);
			for (const predecessor of [original, successor]) {
				await expect(predecessor).toContainText(/replaced/i);
				await expectAuthor(page, predecessor, author);
				await expectReadOnly(predecessor);
			}
			await expectReplacement(page, original, successor, "Original predecessor", "First successor");
			await expectReplacement(page, successor, current, "First successor", "Second successor");
			const chain = await openNoteDetails(page, original);
			await chain.getByRole("link", { name: "View replacement note" }).click();
			await expect(chain.getByRole("heading", { name: "Revised draft", exact: true })).toBeVisible();
			await chain.getByRole("link", { name: "View replacement note" }).click();
			await expect(chain.getByRole("heading", { name: "Final decision", exact: true })).toBeVisible();
			await expect(chain).toContainText("Second successor");
			await cancelDialog(chain);
			const other = await saveNote(page, "Another current note");
			await expect(history(page)).toHaveAttribute("aria-pressed", "true");
			await saveNote(page, "Edited current successor", "Edit", current);
			await expect(history(page)).toHaveAttribute("aria-pressed", "true");
			await expect(original).not.toContainText("Original predecessor");
			await expect(successor).not.toContainText("First successor");
			await deleteNote(page, current);
			await expect(history(page)).toHaveAttribute("aria-pressed", "true");
			await expect(current).toContainText(/deleted/i);
			await expect(current.locator("em")).toHaveText("Deleted");
			await expect(current.locator("em")).toHaveCSS("font-style", "italic");
			await expect(current).not.toContainText("Edited current successor");
			const deletedDetails = await openNoteDetails(page, current);
			await expect(deletedDetails).toContainText("Edited current successor");
			await expect(deletedDetails).toContainText("Deleted");
			await cancelDialog(deletedDetails);
			await expectAuthor(page, current, author);
			await expectReadOnly(current);
			await expect(entries(page)).toHaveCount(4);
			await expectNoOverflow(page);
			await history(page).focus();
			await page.keyboard.press("Space");
			await expect(history(page)).toHaveAttribute("aria-pressed", "false");
			await expect(entries(page)).toHaveCount(1);
			await expect(other).toBeVisible();
			await deleteNote(page, other);
			await expect(entries(page)).toHaveCount(0);
			await expect(history(page)).toBeVisible();
			await page.reload();
			await selectNotes(page);
			await expect(entries(page)).toHaveCount(0);
			await showHistory(page);
			await expect(entries(page)).toHaveCount(4);
			// Do not wait for partial responses between clicks: the final choice must win.
			await history(page).click();
			await history(page).click();
			await history(page).click();
			await expect(history(page)).toHaveAttribute("aria-pressed", "false");
			await expect(entries(page)).toHaveCount(0);
			await showHistory(page);
			await expect(entries(page)).toHaveCount(4);
			await expect(original).toContainText(/replaced/i);
			await expect(current).toContainText(/deleted/i);
		});

		test("Inbox notes survive filing and Trash/restore without reactivating history", async ({ page }) => {
			const { author, browseURL } = await setupDocument(page, true);
			const original = await saveNote(page, "Inbox predecessor");
			const current = await saveNote(page, "Inbox current", "Replace", original);
			await saveNote(page, "Inbox corrected", "Edit", current);
			const deleted = await saveNote(page, "Inbox deleted note");
			await deleteNote(page, deleted);
			await expect(deleted).toHaveCount(0);
			await page.getByRole("tab", { name: "Move", exact: true }).click();
			await page.getByText("Select destination manually", { exact: true }).click();
			const move = page.getByRole("dialog").filter({
				has: page.getByRole("heading", { name: /Move file to/ }),
			});
			await move.getByRole("button", { name: "Save", exact: true }).click();
			await expect(move).not.toBeVisible();
			await page.goto(browseURL);
			await openDocument(page);
			await expectAuthor(page, current, author);
			await expect(entries(page)).toHaveCount(1);
			await page.goto(browseURL);
			// Existing document menus use the contextmenu gesture (also with a mobile viewport).
			const row = page.getByRole("listitem").filter({
				has: page.getByRole("heading", { name: fileName, exact: true }),
			});
			await row.click({ button: "right" });
			page.once("dialog", dialog => dialog.accept());
			await row.getByText("Delete", { exact: true }).click();
			await expect(page.getByRole("heading", { name: fileName, exact: true })).toHaveCount(0);
			await navigateSection(page, "Trash");
			const trashURL = page.url();
			await openDocument(page);
			await expect(entries(page)).toHaveCount(1);
			await expect(page.getByRole("button", { name: "Add note", exact: true })).toHaveCount(0);
			await showHistory(page);
			await expect(entries(page)).toHaveCount(3);
			for (const entry of [original, current, deleted]) {
				await expectAuthor(page, entry, author);
				await expectReadOnly(entry);
			}
			await expect(original).toContainText(/replaced/i);
			await expect(deleted).toContainText(/deleted/i);
			await expectNoOverflow(page);
			await page.goto(trashURL);
			const trashRow = page.locator("#trashList").getByRole("listitem")
				.filter({ hasText: fileName });
			await trashRow.click({ button: "right" });
			page.once("dialog", dialog => dialog.accept());
			await trashRow.getByText("Restore", { exact: true }).click();
			await expect(trashRow).toHaveCount(0);
			await page.goto(browseURL);
			await openDocument(page);
			await expect(entries(page)).toHaveCount(1);
			await expect(current.getByRole("button", { name: "Actions", exact: true })).toBeVisible();
			await showHistory(page);
			await expect(entries(page)).toHaveCount(3);
			await expect(original).toContainText(/replaced/i);
			await expect(deleted).toContainText(/deleted/i);
		});

		test("merges an Inbox replacement chain and deleted notes into an existing target", async ({ page }) => {
			const { author, browseURL, listURL } = await setupDocument(page, true);
			const original = await saveNote(page, "Transferred predecessor");
			const successor = await saveNote(page, "Transferred successor", "Replace", original);
			const current = await saveNote(page, "Transferred current", "Replace", successor);
			const deleted = await saveNote(page, "Transferred deleted");
			await deleteNote(page, deleted);
			await page.goto(browseURL);
			await uploadFileWithToolbar(page, fixturePath("upload-beta.txt"));
			await openDocument(page, "upload-beta.txt");
			const target = await saveNote(page, "Target note stays unchanged");
			await page.getByRole("tab", { name: "Versions", exact: true }).click();
			const openMerge = async () => {
				await page.getByRole("link", { name: /Add new version from inbox/ }).click();
				const dialog = page.locator("#fileVersionFromInboxDialog");
				await expect(dialog).toBeVisible();
				const confirmation = dialog.locator('input[name="ConfirmWarning"]');
				const label = dialog.locator(`label[for="${await confirmation.getAttribute("id")}"]`);
				const originalLabel = await label.textContent();
				// Supplement the real English journey with the reported German wrapping case,
				// without changing the shared fixture account's language.
				for (const text of [originalLabel!,
					"Ich verstehe, dass die Metadaten der Datei aus der Inbox (Dokumenttyp, Tags, " +
					"Felder) beim Zusammenführen verloren gehen. Notizen und ihr Verlauf bleiben erhalten.",
				]) {
					await label.evaluate((element, text) => { element.textContent = text; }, text);
					await expect.poll(() => confirmation.evaluate(input => {
						const control = input.parentElement!.parentElement!;
						const row = control.parentElement!;
						const label = row.querySelector("label")!.getBoundingClientRect();
						const bounds = row.getBoundingClientRect();
						const form = input.closest("form")!.getBoundingClientRect();
						const search = document.querySelector("#fileVersionFromInboxSearch")!
							.getBoundingClientRect();
						return label.top >= Math.max(bounds.top, form.top) - 1 &&
							label.bottom <= bounds.bottom + 1 && label.bottom <= search.top &&
							bounds.height >= 48 && control.getBoundingClientRect().width === 48;
					})).toBe(true);
				}
				await label.evaluate((element, text) => { element.textContent = text; }, originalLabel);
				await expect(confirmation).toHaveAttribute("required", "");
				await confirmation.focus();
				await page.keyboard.press("Space");
				await expect(confirmation).toBeChecked();
				if (device.hasTouch) await label.tap();
				else await label.click();
				await expect(confirmation).not.toBeChecked();
				await dialog.getByRole("heading", { name: fileName, exact: true }).click();
				await dialog.getByRole("checkbox", { name: /metadata.*lost/i }).check();
				return dialog;
			};
			await cancelDialog(await openMerge());
			await selectNotes(page);
			await expect(entries(page)).toHaveCount(1);
			await page.getByRole("tab", { name: "Versions", exact: true }).click();
			const merge = await openMerge();
			await merge.getByRole("button", { name: "Add", exact: true }).click();
			await expect(merge).not.toBeVisible();
			await selectNotes(page);
			await expect(entries(page)).toHaveCount(2);
			await expect(target).toContainText("Target note stays unchanged");
			await expect(current).toContainText("Transferred current");
			await showHistory(page);
			await expect(entries(page)).toHaveCount(5);
			for (const entry of [original, successor, current, deleted, target]) {
				await expectAuthor(page, entry, author);
			}
			await expect(original).toContainText(/replaced/i);
			await expect(successor).toContainText(/replaced/i);
			await expect(deleted).toContainText(/deleted/i);
			await expectReplacement(page, original, successor,
				"Transferred predecessor", "Transferred successor");
			await page.reload();
			await selectNotes(page);
			await showHistory(page);
			await expect(entries(page)).toHaveCount(5);
			await page.goto(listURL);
			await expect(page.getByRole("heading", { name: fileName, exact: true })).toHaveCount(0);
		});
	});
}
