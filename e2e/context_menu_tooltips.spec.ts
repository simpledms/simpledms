import { expect, test, type Locator, type Page } from "@playwright/test";

import { fixturePath, uniqueSuffix } from "./helpers";

async function expandNavigation(page: Page) {
	const toggle = page.getByRole("button", { name: "Open main menu", exact: true }).first();
	if (await toggle.getAttribute("aria-expanded") !== "true") await toggle.click();
}

async function createSpace(page: Page) {
	await page.goto("/dashboard/");
	await expandNavigation(page);
	await page.getByRole("button", { name: /^business / }).click();
	await page.getByRole("link", { name: "Spaces", exact: true }).click();
	await page.getByRole("link", { name: /Create space|^add$/ }).first().click();
	const name = `E2E Overflow ${uniqueSuffix()}`;
	await page.getByRole("textbox", { name: "Name", exact: true }).fill(name);
	await page.getByRole("checkbox", { name: "Add me as space owner" }).check();
	await page.getByRole("button", { name: "Save", exact: true }).click();
	await expect(page.getByRole("dialog")).toHaveCount(0);
	const heading = page.getByRole("heading", { name, exact: true });
	return page.locator("[hx-get]").filter({ has: heading }).last();
}

async function checkMenu(page: Page, owner: Locator, touch: boolean) {
	const trigger = owner.getByRole("button", { name: "Actions", exact: true }).first();
	const menu = owner.getByRole("menu").first();
	const url = page.url();
	const checked = await page.locator("input[type=radio]:checked").evaluateAll(
		inputs => inputs.map(input => input.id),
	);
	if (touch) await trigger.tap();
	else await trigger.click();
	await expect(menu).toBeVisible();
	await expect(trigger).toHaveAttribute("aria-expanded", "true");
	await expect(page).toHaveURL(url);
	expect(await page.locator("input[type=radio]:checked").evaluateAll(
		inputs => inputs.map(input => input.id),
	)).toEqual(checked);
	const rect = await menu.boundingBox();
	expect(rect!.x).toBeGreaterThanOrEqual(0);
	expect(rect!.x + rect!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
	await page.keyboard.press("Escape");
	await expect(menu).not.toBeVisible();
	await expect(trigger).toBeFocused();
	await page.keyboard.press("Enter");
	await expect(menu).toBeVisible();
	await page.keyboard.press("Escape");
	await owner.getByRole("heading").first().click({ button: "right" });
	await expect(menu).toBeVisible();
	await page.keyboard.press("Escape");
	await expect(trigger).toHaveAttribute("aria-expanded", "false");
}

for (const touch of [false, true]) {
	test.describe(touch ? "mobile overflow" : "desktop overflow", () => {
		test.use({
			storageState: "e2e/.auth/admin.json",
			viewport: touch ? { width: 390, height: 844 } : { width: 1440, height: 1000 },
			isMobile: touch,
			hasTouch: touch,
		});

		test("cards, nested lists and radio rows retain their primary actions", async ({ page }) => {
			const errors: string[] = [];
			page.on("pageerror", error => errors.push(error.message));
			const card = await createSpace(page);
			await checkMenu(page, card, touch);
			await card.getByRole("button", { name: "Actions", exact: true }).click();
			await card.getByRole("link", { name: "label Tags", exact: true }).click();

			await page.getByRole("heading", { name: "Create new tag or group", exact: true }).click();
			await page.getByRole("textbox", { name: "Name", exact: true }).fill("Parent group");
			await page.getByRole("combobox", { name: "Type", exact: true }).selectOption({ label: "Group" });
			await page.getByRole("button", { name: "Save", exact: true }).click();
			const group = page.getByRole("listitem").filter({
				has: page.getByRole("heading", { name: "Parent group", exact: true }),
			});
			await checkMenu(page, group, touch);
			await expect(group.locator("details")).not.toHaveAttribute("open");
			await group.getByRole("heading", { name: "Parent group", exact: true }).click();
			await group.getByRole("heading", { name: "Create new tag", exact: true }).click();
			await page.getByRole("textbox", { name: "Name", exact: true }).fill("Child tag");
			await page.getByRole("combobox", { name: "Type", exact: true }).selectOption({ label: "Simple" });
			await page.getByRole("button", { name: "Save", exact: true }).click();
			const child = group.getByRole("listitem").filter({
				has: page.getByRole("heading", { name: "Child tag", exact: true }),
			});
			await checkMenu(page, child, touch);
			await expect(group.locator("details").first()).toHaveAttribute("open");
			await child.getByRole("button", { name: "Actions", exact: true }).click();
			await child.getByRole("link", { name: "edit Edit", exact: true }).click();
			await page.getByRole("textbox", { name: "Name", exact: true }).fill("Renamed child");
			await page.getByRole("button", { name: "Save", exact: true }).click();
			await expect(group.getByRole("heading", { name: "Renamed child", exact: true })).toBeVisible();

			await expandNavigation(page);
			await page.getByRole("link", { name: "Files", exact: true }).click();
			if (!touch) await expandNavigation(page);
			await page.getByRole("link", { name: /Upload file/ }).first().click();
			const upload = page.getByRole("dialog").filter({ hasText: "File upload" });
			await upload.locator("input[type=file]").first().setInputFiles([
				fixturePath("upload-alpha.txt"), fixturePath("upload-beta.txt"),
			]);
			await expect(upload.getByText("Upload complete")).toBeVisible();
			await upload.getByRole("button", { name: "close", exact: true }).click();
			const file = page.getByRole("listitem").filter({
				has: page.getByRole("heading", { name: "upload-alpha.txt", exact: true }),
			});
			await checkMenu(page, file, touch);
			const unselectedColor = await file.getByRole("button", { name: "Actions", exact: true })
				.locator("i").evaluate(element => getComputedStyle(element).color);
			// Exercise native radio selection before navigation replaces the list on mobile.
			await file.locator('input[type=radio]').evaluate((input: HTMLInputElement) => input.click());
			await expect(file.locator('input[type=radio]')).toBeChecked();
			const selectedColor = await file.getByRole("heading").evaluate(
				element => getComputedStyle(element).color,
			);
			const actions = file.getByRole("button", { name: "Actions", exact: true });
			await expect(actions.locator("i")).toHaveCSS("color", selectedColor);
			await actions.focus();
			await expect(actions.locator("i")).toHaveCSS("color", selectedColor);
			if (!touch) {
				await actions.hover();
				await expect(actions.locator("div").first()).toHaveCSS(
					"background-color", selectedColor.replace("rgb(", "rgba(").replace(")", ", 0.08)"),
				);
			}
			const otherFile = page.getByRole("listitem").filter({
				has: page.getByRole("heading", { name: "upload-beta.txt", exact: true }),
			});
			await otherFile.locator('input[type=radio]')
				.evaluate((input: HTMLInputElement) => input.click());
			await expect(otherFile.locator('input[type=radio]')).toBeChecked();
			await expect(file.getByRole("button", { name: "Actions", exact: true }).locator("i"))
				.toHaveCSS("color", unselectedColor);
			await file.getByRole("heading", { name: "upload-alpha.txt", exact: true }).click();
			await expect(page).toHaveURL(/\/browse\/[^/]+\//);
			expect(errors).toEqual([]);
		});
	});
}

test.describe("plain tooltips", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });

	test("hover, keyboard, Escape, modal and HTMX lifecycle", async ({ page }) => {
		const card = await createSpace(page);
		const trigger = card.getByRole("button", { name: "Actions", exact: true });
		// Finish scrolling before entering the control: scrolling dismisses tooltips.
		await trigger.scrollIntoViewIfNeeded();
		await page.mouse.move(0, 0);
		await trigger.hover();
		const tooltip = page.getByRole("tooltip");
		await expect(tooltip).toHaveText("Actions");
		await expect(tooltip).toBeVisible();
		await expect(trigger).not.toHaveAttribute("title");
		await expect(trigger).toHaveAttribute("aria-describedby", "app-tooltip");
		await tooltip.hover();
		await expect(tooltip).toBeVisible();
		await page.keyboard.press("Escape");
		await expect(tooltip).not.toBeVisible();
		await expect(trigger).not.toHaveAttribute("aria-describedby");
		await trigger.focus();
		await page.keyboard.press("Tab");
		await page.keyboard.press("Shift+Tab");
		await expect(trigger).toBeFocused();
		await expect(tooltip).toBeVisible();
		await trigger.click();
		await expect(tooltip).not.toBeVisible();
		await page.getByRole("menu").filter({ visible: true })
			.getByRole("link", { name: "edit Edit", exact: true }).click();
		const dialog = page.getByRole("dialog");
		await expect(dialog).toBeVisible();
		// Exercise unusual text and pre-existing descriptions on a real dialog control.
		const input = dialog.getByRole("textbox", { name: "Name", exact: true });
		await input.evaluate(element => {
			element.setAttribute("data-tooltip", "Created: Today\nAuthor: <b>Not markup</b>");
			element.setAttribute("aria-describedby", "existing-help");
		});
		await input.hover();
		await expect(tooltip).toHaveText("Created: Today\nAuthor: <b>Not markup</b>");
		await expect(tooltip.locator("b")).toHaveCount(0);
		await expect(input).toHaveAttribute("aria-describedby", "existing-help app-tooltip");
		await expect(dialog.locator("#app-tooltip")).toBeVisible();
		await page.keyboard.press("Escape");
		await expect(dialog).toBeVisible();
		await expect(input).toHaveAttribute("aria-describedby", "existing-help");
		await dialog.getByRole("button", { name: "Save", exact: true }).click();
		await expect(dialog).not.toBeVisible();
		await trigger.hover();
		await expect(tooltip).toBeVisible();
		await page.setViewportSize({ width: 390, height: 844 });
		await expect(tooltip).not.toBeVisible();
		await trigger.scrollIntoViewIfNeeded();
		await page.mouse.move(0, 0);
		await trigger.hover();
		await expect(tooltip).toBeVisible();
		const rect = await tooltip.boundingBox();
		expect(rect!.x).toBeGreaterThanOrEqual(8);
		expect(rect!.x + rect!.width).toBeLessThanOrEqual(382);
	});
});
