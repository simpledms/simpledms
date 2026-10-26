import { expect, test, type Page } from "@playwright/test";

import { createSpaceAndSelect, fixturePath, uniqueSuffix } from "./helpers";

const sessionCookieName = "simpledms_session";

async function shareTarget(page: Page) {
	const manifestURL = await page.locator('link[rel="manifest"]').getAttribute("href");
	expect(manifestURL).toBeTruthy();
	const response = await page.request.get(manifestURL!);
	expect(response.ok()).toBeTruthy();
	const manifest = await response.json();
	expect(manifest.share_target).toMatchObject({
		method: "POST",
		enctype: "multipart/form-data",
	});
	const fileParam = manifest.share_target.params.files[0];
	expect(fileParam.name).toBe("file");
	return {
		action: new URL(manifest.share_target.action, page.url()).href,
		fileParamName: fileParam.name as string,
	};
}

async function attachSharedFile(
	page: Page,
	action: string,
	fileParamName: string,
) {
	await page.evaluate(({ action, fileParamName }) => {
		const form = document.createElement("form");
		form.method = "POST";
		form.enctype = "multipart/form-data";
		form.action = action;
		form.id = "native-share-form";
		const input = document.createElement("input");
		input.type = "file";
		input.name = fileParamName;
		form.append(input);
		document.body.append(form);
	}, { action, fileParamName });
	await page.locator('#native-share-form input[type="file"]').setInputFiles(fixturePath("upload-alpha.txt"));
}

async function chooseSpaceAndVerifyInbox(page: Page, spaceName: string) {
	await expect(page.getByText("Select Space", { exact: true })).toBeVisible();
	await page.getByRole("heading", { name: spaceName, exact: true }).click();
	await expect(page).toHaveURL(/\/space\/[^/]+\/inbox\/$/);
	await expect(page.getByRole("heading", { name: "upload-alpha.txt", exact: true })).toBeVisible();
}

async function serveExternalShareSource(
	page: Page,
	action: string,
	onShareRequest: (cookieHeader: string) => void,
) {
	await page.route(action, async route => {
		const headers = await route.request().allHeaders();
		onShareRequest(headers.cookie ?? "");
		// This test adapter models OS share-target metadata without altering the
		// browser-computed Cookie header; it is not a reproduction of native OS behavior.
		await route.continue({
			headers: {
				...headers,
				origin: "null",
				"sec-fetch-site": "none",
			},
		});
	});
	await page.route("https://share-source.invalid/**", route => route.fulfill({
		status: 200,
		contentType: "text/html",
		body: "<!doctype html><title>External share source</title>",
	}));
	await page.goto("https://share-source.invalid/share");
}

test.describe("native file sharing", () => {
	test.use({ storageState: "e2e/.auth/admin.json" });
	test.setTimeout(60_000);

	test("mobile share target uploads a file and opens it in the selected Inbox", async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		const spaceName = `E2E Native Share ${uniqueSuffix()}`;
		await createSpaceAndSelect(page, spaceName);
		const target = await shareTarget(page);
		await attachSharedFile(page, target.action, target.fileParamName);

		await Promise.all([
			page.waitForURL(/\/open-file\/select-space\//),
			page.locator("#native-share-form").evaluate((form: HTMLFormElement) => form.requestSubmit()),
		]);
		await chooseSpaceAndVerifyInbox(page, spaceName);
	});

	test("cross-site share carries the production session and reaches the selected Inbox", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		const spaceName = `E2E Native Share Secure ${uniqueSuffix()}`;
		await createSpaceAndSelect(page, spaceName);

		// Read the app-issued cookie as-is: never make the regression pass by injecting
		// a SameSite=None cookie into the browser context.
		const sessionCookie = (await page.context().cookies(page.url())).find(
			cookie => cookie.name === sessionCookieName,
		);
		expect(sessionCookie, "authenticated app session cookie should exist").toBeTruthy();
		if (!sessionCookie!.secure) {
			test.skip(true, "requires production secure cookies; insecure dev retains Lax");
		}
		expect(sessionCookie!.sameSite).toBe("None");

		const target = await shareTarget(page);
		const targetURL = new URL(target.action);
		let browserSentSession = false;
		expect(targetURL.origin).not.toBe("https://share-source.invalid");
		await serveExternalShareSource(page, target.action, cookieHeader => {
			browserSentSession = cookieHeader.split("; ").some(cookie =>
				cookie.startsWith(`${sessionCookieName}=`),
			);
		});
		await attachSharedFile(page, target.action, target.fileParamName);
		await Promise.all([
			page.waitForURL(/\/open-file\/select-space\//),
			page.locator("#native-share-form").evaluate((form: HTMLFormElement) => form.requestSubmit()),
		]);
		expect(browserSentSession, "browser should send the actual session cookie cross-site").toBe(true);
		await chooseSpaceAndVerifyInbox(page, spaceName);
	});

	test("a SameSite=Lax session reproduces the cross-site share redirect to the dashboard", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		const spaceName = `E2E Native Share Lax Control ${uniqueSuffix()}`;
		await createSpaceAndSelect(page, spaceName);

		const sessionCookie = (await page.context().cookies(page.url())).find(
			cookie => cookie.name === sessionCookieName,
		);
		expect(sessionCookie, "authenticated app session cookie should exist").toBeTruthy();
		if (!sessionCookie!.secure) {
			test.skip(true, "requires production secure cookies; insecure dev retains Lax");
		}
		const target = await shareTarget(page);
		const targetURL = new URL(target.action);
		expect(targetURL.origin).not.toBe("https://share-source.invalid");

		// Deliberately install the old/broken policy only for this negative control.
		// This is not used by the positive test, which tests the untouched app cookie.
		await page.context().addCookies([{ ...sessionCookie!, sameSite: "Lax" }]);
		let browserSentSession = true;
		await serveExternalShareSource(page, target.action, cookieHeader => {
			browserSentSession = cookieHeader.split("; ").some(cookie =>
				cookie.startsWith(`${sessionCookieName}=`),
			);
		});
		await attachSharedFile(page, target.action, target.fileParamName);
		await Promise.all([
			page.waitForResponse(response =>
				response.url() === target.action && response.request().method() === "POST",
			),
			page.waitForResponse(response => {
				const url = new URL(response.url());
				return url.origin === targetURL.origin && url.pathname === "/"
					&& response.request().method() === "GET";
			}),
			page.waitForURL(/\/dashboard\/$/),
			page.locator("#native-share-form").evaluate((form: HTMLFormElement) => form.requestSubmit()),
		]).then(([shareResponse, rootResponse]) => {
			expect(shareResponse.status()).toBe(303);
			expect(shareResponse.headers().location).toBe("/");
			expect(rootResponse.status()).toBe(303);
			expect(rootResponse.headers().location).toMatch(/\/dashboard\/$/);
		});
		expect(browserSentSession, "browser should omit the SameSite=Lax cookie on cross-site POST").toBe(false);
		await expect(page.getByText("Select Space", { exact: true })).toHaveCount(0);
		await expect(page.getByRole("heading", { name: spaceName, exact: true })).toHaveCount(0);
	});
});
