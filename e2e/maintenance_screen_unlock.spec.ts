import { expect, test, type Request, type Response, type Route } from "@playwright/test";

const correctPassphrase = process.env.E2E_MAINTENANCE_PASSPHRASE ?? "";
const wrongPassphrase = "e2e-wrong-maintenance-passphrase-4d91";
const maintenanceMarker = "x-simpledms-maintenance";

test.describe("maintenance screen unlock", () => {
	test.describe.configure({ mode: "serial" });

	test("unlocks without leaking credentials", async ({ page }) => {
		test.skip(
			correctPassphrase === "",
			"Set E2E_MAINTENANCE_PASSPHRASE for the locked instance.",
		);
		expect(
			correctPassphrase === wrongPassphrase,
			"configured and invalid passphrases must differ",
		).toBe(false);

		const sentinels = [correctPassphrase, wrongPassphrase];
		const networkChecks: Promise<void>[] = [];
		const observedPostBodies = new Set<string>();

		const requestHandler = (request: Request) => {
			networkChecks.push((async () => {
				const headers = JSON.stringify(await request.allHeaders());
				for (const sentinel of sentinels) {
					expect(
						request.url().includes(sentinel),
						"request URL exposed a passphrase",
					).toBe(false);
					expect(
						headers.includes(sentinel),
						"request headers exposed a passphrase",
					).toBe(false);
				}

				const postData = request.postData() ?? "";
				for (const sentinel of sentinels) {
					if (!postData.includes(sentinel)) {
						continue;
					}
					expect(new URL(request.url()).pathname).toBe("/-/unlock-cmd");
					expect(request.method()).toBe("POST");
					observedPostBodies.add(sentinel);
				}
			})());
		};

		const responseHandler = (response: Response) => {
			networkChecks.push((async () => {
				const headers = JSON.stringify(await response.allHeaders());
				const body = await response.body().catch(() => Buffer.from(""));
				for (const sentinel of sentinels) {
					expect(
						response.url().includes(sentinel),
						"response URL exposed a passphrase",
					).toBe(false);
					expect(
						headers.includes(sentinel),
						"response headers exposed a passphrase",
					).toBe(false);
					expect(
						body.toString().includes(sentinel),
						"response body exposed a passphrase",
					).toBe(false);
				}
			})());
		};
		page.on("request", requestHandler);
		page.on("response", responseHandler);

		const localizedCases = [
			{
				language: "de",
				label: "Anwendungspassphrase",
				required: "Passphrase ist erforderlich.",
				invalid: "Ungültige Passphrase.",
			},
			{
				language: "fr",
				label: "Phrase secrète de l’application",
				required: "La phrase secrète est requise.",
				invalid: "Phrase secrète invalide.",
			},
			{
				language: "it",
				label: "Passphrase dell’applicazione",
				required: "La passphrase è obbligatoria.",
				invalid: "Passphrase non valida.",
			},
		];
		for (const localizedCase of localizedCases) {
			const languageHandler = async (route: Route) => {
				await route.continue({
					headers: {
						...route.request().headers(),
						"accept-language": localizedCase.language,
					},
				});
			};
			await page.route("**/*", languageHandler);
			await page.goto("/?unlock");
			await expect(page.locator("html"))
				.toHaveAttribute("lang", localizedCase.language);
			const localizedInput = page.getByLabel(localizedCase.label);
			const localizedAlert = page.getByRole("alert");

			await localizedInput.press("Enter");
			await expect(localizedAlert).toHaveText(localizedCase.required);
			await expect(localizedInput).toHaveAttribute("aria-invalid", "true");
			await expect(localizedInput).toBeFocused();

			await localizedInput.fill(wrongPassphrase);
			await localizedInput.press("Enter");
			await expect(localizedAlert).toHaveText(localizedCase.invalid);
			await expect(localizedInput).toHaveAttribute("aria-invalid", "true");
			await expect(localizedInput).toHaveValue("");
			await expect(localizedInput).toBeFocused();
			await page.unroute("**/*", languageHandler);
		}

		const historyURL = "/?history=before";
		const historyResponse = await page.goto(historyURL);
		expect(historyResponse?.status()).toBe(503);
		expect(historyResponse?.headers()[maintenanceMarker]).toBe("true");
		await expect(page.getByRole("heading", { name: /Maintenance mode/ })).toBeVisible();
		await expect(page.getByLabel("Application passphrase")).toHaveCount(0);

		const unlockURL = "/?keep=1&unlock=false&unlock=again#details";
		const unlockResponse = await page.goto(unlockURL);
		expect(unlockResponse?.status()).toBe(503);
		expect(unlockResponse?.headers()[maintenanceMarker]).toBe("true");
		expect(unlockResponse?.headers()["cache-control"]).toBe("no-store");
		expect(unlockResponse?.headers().pragma).toBe("no-cache");

		const input = page.getByLabel("Application passphrase");
		const form = page.locator('form[action="/-/unlock-cmd"]');
		const submit = page.getByRole("button", { name: "Unlock application" });
		const status = page.getByRole("status");
		const alert = page.getByRole("alert");
		await expect(input).toHaveAttribute("type", "password");
		await expect(input).toHaveAttribute("required", "");

		await page.setViewportSize({ width: 360, height: 640 });
		await expect(input).toBeInViewport();
		await expect(submit).toBeInViewport();
		await input.press("Enter");
		await expect(alert).toHaveText("Passphrase is required.");
		await expect(input).toBeFocused();
		await page.setViewportSize({ width: 1280, height: 720 });

		await page.route("**/-/unlock-cmd", async (route) => {
			await route.fulfill({ status: 500, body: "internal details must stay hidden" });
		}, { times: 1 });
		await input.fill(wrongPassphrase);
		await input.press("Enter");
		await expect(alert).toHaveText("Something went wrong. Please try again.");
		await expect(alert).not.toContainText("internal details");
		await expect(input).toHaveValue("");
		await expect(input).toBeFocused();
		expect(
			(await page.content()).includes(wrongPassphrase),
			"rendered HTML exposed a passphrase",
		).toBe(false);

		await page.route("**/-/unlock-cmd", async (route) => {
			await route.fulfill({ status: 429, body: "rate-limit details must stay hidden" });
		}, { times: 1 });
		await input.fill(wrongPassphrase);
		await input.press("Enter");
		await expect(alert)
			.toHaveText("Too many unlock attempts. Please try again later.");
		await expect(alert).not.toContainText("rate-limit details");
		await expect(input).toHaveValue("");
		await expect(input).toBeFocused();

		let releaseInvalidRequest = () => {};
		const invalidRequestReleased = new Promise<void>((resolve) => {
			releaseInvalidRequest = resolve;
		});
		let markInvalidRequestStarted = () => {};
		const invalidRequestStarted = new Promise<void>((resolve) => {
			markInvalidRequestStarted = resolve;
		});
		let invalidRequestCount = 0;
		const delayedInvalidHandler = async (route: Route) => {
			invalidRequestCount++;
			if (invalidRequestCount === 1) {
				markInvalidRequestStarted();
				await invalidRequestReleased;
			}
			await route.continue();
		};
		await page.route("**/-/unlock-cmd", delayedInvalidHandler);

		await input.fill(wrongPassphrase);
		const invalidSubmit = input.press("Enter");
		await invalidRequestStarted;
		await expect(submit).toBeDisabled();
		await expect(form).toHaveAttribute("aria-busy", "true");
		await expect(status).toHaveText("");
		await input.press("Enter");
		await submit.evaluate((button: HTMLButtonElement) => button.click());
		releaseInvalidRequest();
		await invalidSubmit;
		await expect(alert).toHaveText("Invalid passphrase.");
		await expect(input).toHaveValue("");
		await expect(input).toBeFocused();
		await expect(submit).toBeEnabled();
		expect(invalidRequestCount).toBe(1);
		await page.unroute("**/-/unlock-cmd", delayedInvalidHandler);
		expect(
			(await page.content()).includes(wrongPassphrase),
			"rendered HTML exposed a passphrase",
		).toBe(false);

		const cleanedURL = new URL("/?keep=1#details", page.url());
		let networkFailurePolls = 0;
		let markedPolls = 0;
		let markerFree503Polls = 0;
		await page.route((url) => {
			return url.pathname === cleanedURL.pathname && url.search === cleanedURL.search;
		}, async (route) => {
			const request = route.request();
			if (request.resourceType() !== "fetch") {
				await route.continue();
				return;
			}
			if (networkFailurePolls === 0) {
				networkFailurePolls++;
				await route.abort("connectionfailed");
				return;
			}
			if (markedPolls === 0) {
				markedPolls++;
				await route.fulfill({
					status: 503,
					headers: { "X-SimpleDMS-Maintenance": "true" },
					body: "still starting",
				});
				return;
			}

			try {
				const response = await route.fetch();
				if (response.headers()[maintenanceMarker] === "true") {
					markedPolls++;
					await route.fulfill({ response });
					return;
				}
				markerFree503Polls++;
				await route.fulfill({ status: 503, body: "normal route response" });
			} catch (_) {
				await route.abort("connectionfailed");
			}
		});

		await input.fill(correctPassphrase);
		await input.press("Enter");
		await expect(input).toHaveValue("");
		await expect(submit).toBeDisabled();
		await expect(form).toHaveAttribute("aria-busy", "false");
		await expect(status).toHaveText("Application unlocked. Starting up.");
		await expect(page).toHaveURL(cleanedURL.href, { timeout: 30_000 });
		expect(networkFailurePolls).toBe(1);
		expect(markedPolls).toBeGreaterThan(0);
		expect(markerFree503Polls).toBe(1);

		const finalContent = await page.content();
		for (const sentinel of sentinels) {
			expect(
				page.url().includes(sentinel),
				"final URL exposed a passphrase",
			).toBe(false);
			expect(
				finalContent.includes(sentinel),
				"rendered HTML exposed a passphrase",
			).toBe(false);
		}

		await page.goBack();
		await expect(page).toHaveURL(new URL(historyURL, cleanedURL).href);
		expect(page.url().includes("unlock")).toBe(false);
		for (const sentinel of sentinels) {
			expect(
				page.url().includes(sentinel),
				"browser history exposed a passphrase",
			).toBe(false);
		}
		await page.waitForLoadState("load");
		page.off("request", requestHandler);
		page.off("response", responseHandler);
		await Promise.all(networkChecks);
		for (const sentinel of sentinels) {
			expect(observedPostBodies.has(sentinel)).toBe(true);
		}
	});
});
