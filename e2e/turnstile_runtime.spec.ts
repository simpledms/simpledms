import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

declare global {
	interface Window {
		testTurnstile: { options: Record<string, (token?: string) => void>; id: string };
		testRequestPrevented: boolean;
		testRetryPrevented: boolean;
		testReset: string;
		testRemoved: string;
	}
}

const widgetTemplate = readFileSync(
	resolve(__dirname, "../core/ui/widget/turnstile.gohtml"),
	"utf8",
);
const widgetScript = widgetTemplate.match(/<script>([\s\S]*?)<\/script>/)?.[1];
if (!widgetScript) throw new Error("Turnstile widget script not found");

const snackbarRuntime = readFileSync(
	resolve(__dirname, "../ui/uix/web/assets/snackbar_runtime.js"),
	"utf8",
);

const messages = {
	error: "Verification failed. Please try again.",
	wait: "Please wait for verification to complete.",
	expired: "Verification expired. Please try again.",
	unavailable: "Verification is temporarily unavailable. Please try again.",
	submitting: "Registration is being submitted. Please wait.",
	timeout: "Verification timed out. Please try again.",
};

async function mountWidget(page: import("@playwright/test").Page, providerFails = false) {
	await page.route("https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit", async route => {
		if (providerFails) {
			await route.abort();
			return;
		}
		await route.fulfill({
			contentType: "application/javascript",
			body: `window.turnstile = {
				ready() { throw new Error("Remove async/defer before using ready()"); },
				render(target, options) {
					window.testTurnstile = { options, id: "widget-1" };
					const token = document.createElement("input");
					token.type = "hidden";
					token.name = "cf-turnstile-response";
					target.append(token);
					return "widget-1";
				},
				reset(id) { window.testReset = id; },
				remove(id) { window.testRemoved = id; }
			};`,
		});
	});

	await page.setContent(`<!doctype html><form><simpledms-turnstile id="challenge"
		data-sitekey="test-site" data-action="register"
		data-error-message="${messages.error}" data-wait-message="${messages.wait}"
		data-expired-message="${messages.expired}" data-unavailable-message="${messages.unavailable}"
		data-submitting-message="${messages.submitting}" data-timeout-message="${messages.timeout}"
		style="display:block"><div data-challenge></div></simpledms-turnstile>
		<button type="submit">Register</button></form>`);
	await page.addScriptTag({ content: snackbarRuntime });
	await page.addScriptTag({ content: widgetScript });
}

test("loads Cloudflare asynchronously without calling ready and guards requests until token", async ({ page }) => {
	await mountWidget(page);
	await expect.poll(() => page.evaluate(() => Boolean(window.testTurnstile))).toBe(true);
	await expect(page.locator('input[name="cf-turnstile-response"]')).toHaveCount(1);
	await page.locator("form").evaluate(form => {
		const event = new CustomEvent("htmx:beforeRequest", { cancelable: true, detail: { elt: form } });
		form.dispatchEvent(event);
		window.testRequestPrevented = event.defaultPrevented;
	});
	await expect(page.locator(".js-snackbar")).toContainText(messages.wait);
	await expect(page.locator(".js-snackbar")).toBeVisible();
	await expect(page.evaluate(() => window.testRequestPrevented)).resolves.toBe(true);
	await page.evaluate(() => window.testTurnstile.options.callback("token"));
	await expect(page.locator(".js-snackbar")).toHaveCount(0);
	await page.locator("form").evaluate(form => {
		const event = new CustomEvent("htmx:beforeRequest", { cancelable: true, detail: { elt: form } });
		form.dispatchEvent(event);
		window.testRequestPrevented = event.defaultPrevented;
	});
	await expect(page.evaluate(() => window.testRequestPrevented)).resolves.toBe(false);
	await page.locator("form").evaluate(form => {
		const event = new CustomEvent("htmx:beforeRequest", { cancelable: true, detail: { elt: form } });
		form.dispatchEvent(event);
		window.testRequestPrevented = event.defaultPrevented;
	});
	await expect(page.locator(".js-snackbar")).toContainText(messages.submitting);
	await expect(page.locator(".js-snackbar")).toBeVisible();
	await expect(page.evaluate(() => window.testRequestPrevented)).resolves.toBe(true);
	await page.locator("form").evaluate(form => form.dispatchEvent(
		new CustomEvent("htmx:afterRequest", { detail: { elt: form } }),
	));
	await expect(page.evaluate(() => window.testReset)).resolves.toBe("widget-1");
});

test("shows provider callback messages and allows a fresh token after request failure", async ({ page }) => {
	await mountWidget(page);
	await expect.poll(() => page.evaluate(() => Boolean(window.testTurnstile))).toBe(true);
	for (const [callback, message] of [
		["expired-callback", messages.expired],
		["error-callback", messages.error],
		["timeout-callback", messages.timeout],
	] as const) {
		await page.evaluate(({ callback }) => window.testTurnstile.options[callback](), { callback });
		await expect(page.locator(".js-snackbar")).toContainText(message);
	}
	await page.evaluate(() => window.testTurnstile.options.callback("token"));
	await page.locator("form").evaluate(form => form.dispatchEvent(
		new CustomEvent("htmx:beforeRequest", { cancelable: true, detail: { elt: form } }),
	));
	await page.locator("form").evaluate(form => form.dispatchEvent(
		new CustomEvent("htmx:afterRequest", { detail: { elt: form } }),
	));
	await page.evaluate(() => window.testTurnstile.options.callback("retry-token"));
	await page.locator("form").evaluate(form => {
		const event = new CustomEvent("htmx:beforeRequest", { cancelable: true, detail: { elt: form } });
		form.dispatchEvent(event);
		window.testRetryPrevented = event.defaultPrevented;
	});
	await expect(page.evaluate(() => window.testRetryPrevented)).resolves.toBe(false);
});

test("reports provider load failure through the snackbar and removes disconnected widgets", async ({ page }) => {
	await mountWidget(page, true);
	await expect(page.locator(".js-snackbar")).toContainText(messages.unavailable);
	await page.locator("#challenge").evaluate(element => element.remove());
	await mountWidget(page);
	await expect.poll(() => page.evaluate(() => Boolean(window.testTurnstile))).toBe(true);
	await page.locator("#challenge").evaluate(element => element.remove());
	await expect(page.evaluate(() => window.testRemoved)).resolves.toBe("widget-1");
});
