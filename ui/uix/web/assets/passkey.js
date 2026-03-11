(function() {
	function base64URLToArrayBuffer(base64url) {
		const padding = "=".repeat((4 - (base64url.length % 4)) % 4)
		const base64 = (base64url + padding).replace(/-/g, "+").replace(/_/g, "/")
		const binary = atob(base64)
		const bytes = new Uint8Array(binary.length)

		for (let i = 0; i < binary.length; i++) {
			bytes[i] = binary.charCodeAt(i)
		}

		return bytes.buffer
	}

	function arrayBufferToBase64URL(buffer) {
		const bytes = new Uint8Array(buffer)
		let binary = ""

		for (let i = 0; i < bytes.byteLength; i++) {
			binary += String.fromCharCode(bytes[i])
		}

		return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "")
	}

	function toJSONCredential(credential) {
		const response = {
			clientDataJSON: arrayBufferToBase64URL(credential.response.clientDataJSON),
		}

		if (credential.response.attestationObject) {
			response.attestationObject = arrayBufferToBase64URL(credential.response.attestationObject)
		}

		if (credential.response.authenticatorData) {
			response.authenticatorData = arrayBufferToBase64URL(credential.response.authenticatorData)
		}

		if (credential.response.signature) {
			response.signature = arrayBufferToBase64URL(credential.response.signature)
		}

		if (credential.response.userHandle) {
			response.userHandle = arrayBufferToBase64URL(credential.response.userHandle)
		}

		return {
			id: credential.id,
			rawId: arrayBufferToBase64URL(credential.rawId),
			type: credential.type,
			response: response,
			clientExtensionResults: credential.getClientExtensionResults ? credential.getClientExtensionResults() : {},
			authenticatorAttachment: credential.authenticatorAttachment,
		}
	}

	function decodeAssertionOptions(options) {
		const decoded = JSON.parse(JSON.stringify(options))
		decoded.challenge = base64URLToArrayBuffer(decoded.challenge)

		if (Array.isArray(decoded.allowCredentials)) {
			decoded.allowCredentials = decoded.allowCredentials.map((credential) => {
				return {
					...credential,
					id: base64URLToArrayBuffer(credential.id),
				}
			})
		}

		return decoded
	}

	function decodeCreationOptions(options) {
		const decoded = JSON.parse(JSON.stringify(options))
		decoded.challenge = base64URLToArrayBuffer(decoded.challenge)
		decoded.user.id = base64URLToArrayBuffer(decoded.user.id)

		if (Array.isArray(decoded.excludeCredentials)) {
			decoded.excludeCredentials = decoded.excludeCredentials.map((credential) => {
				return {
					...credential,
					id: base64URLToArrayBuffer(credential.id),
				}
			})
		}

		return decoded
	}

	async function passkeyRequest(url, payload = {}) {
		const response = await fetch(url, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			credentials: "same-origin",
			body: JSON.stringify(payload),
		})

		const contentType = response.headers.get("content-type") || ""

		if (!response.ok) {
			let errMsg = `Passkey request failed (${response.status}).`
			if (contentType.includes("application/json")) {
				const json = await response.json().catch(() => ({}))
				if (json && typeof json.message === "string" && json.message.length > 0) {
					errMsg = json.message
				}
			} else {
				const text = await response.text().catch(() => "")
				if (text.length > 0) {
					errMsg = text
				}
			}
			throw new Error(errMsg)
		}

		if (contentType.includes("application/json")) {
			return response.json()
		}

		return {}
	}

	function ensurePasskeySupport() {
		if (!window.PublicKeyCredential || !navigator.credentials) {
			throw new Error("Passkeys are not supported in this browser.")
		}
	}

	function dispatchAccountUpdated() {
		document.body.dispatchEvent(
			new CustomEvent("accountUpdated", {
				bubbles: true,
			}),
		)
	}

	function passkeyRuntimeMessage(key, fallback) {
		var messages = document.getElementById("passkey-runtime-messages")
		if (!messages || !messages.dataset) {
			return fallback
		}

		var value = messages.dataset[key]
		if (typeof value === "string" && value.length > 0) {
			return value
		}

		return fallback
	}

	window.simpledmsShowRecoveryCodes = function(recoveryCodesToken) {
		const htmxInstance = window.htmx
		if (!htmxInstance || typeof htmxInstance.ajax !== "function") {
			return Promise.reject(new Error("The backup codes dialog is unavailable. Please reload the page and try again."))
		}

		const existingDialog = document.getElementById("passkeyRecoveryCodesDialog")
		if (existingDialog) {
			existingDialog.close()
			existingDialog.remove()
		}

		return htmxInstance
			.ajax("POST", "/-/auth/passkey-recovery-codes-dialog?wrapper=dialog", {
				target: document.body,
				swap: "beforeend",
				values: {
					Token: recoveryCodesToken,
				},
			})
			.then(() => {
				const dialog = document.getElementById("passkeyRecoveryCodesDialog")
				if (!dialog) {
					throw new Error("The backup codes dialog is unavailable.")
				}

				return new Promise((resolve) => {
					dialog.addEventListener(
						"close",
						() => {
							resolve()
						},
						{
							once: true,
						},
					)
				})
			})
	}

	window.simpledmsPasskeySignIn = async function(event) {
		if (event) {
			event.preventDefault()
		}

		try {
			ensurePasskeySupport()

			const beginPayload = await passkeyRequest("/-/auth/passkey-sign-in-begin-cmd")
			const assertion = await navigator.credentials.get({
				publicKey: decodeAssertionOptions(beginPayload.options.publicKey),
			})

			if (!assertion) {
				throw new Error("Passkey sign-in was cancelled.")
			}

			const finishPayload = await passkeyRequest("/-/auth/passkey-sign-in-finish-cmd", {
				challengeId: beginPayload.challengeId,
				credential: toJSONCredential(assertion),
			})

			if (finishPayload.redirect) {
				window.location.assign(finishPayload.redirect)
				return
			}

			window.location.reload()
		} catch (err) {
			window.simpledmsShowClientSnackbar(err instanceof Error ? err.message : "Passkey sign-in failed.")
		}
	}

	window.simpledmsPasskeyRegister = async function(passkeyName = "", dialogID = "") {
		try {
			ensurePasskeySupport()

			const beginPayload = await passkeyRequest("/-/auth/passkey-register-begin-cmd")

			const attestation = await navigator.credentials.create({
				publicKey: decodeCreationOptions(beginPayload.options.publicKey),
			})

			if (!attestation) {
				throw new Error("Passkey registration was cancelled.")
			}

			const finishPayload = await passkeyRequest("/-/auth/passkey-register-finish-cmd", {
				challengeId: beginPayload.challengeId,
				credential: toJSONCredential(attestation),
				name: typeof passkeyName === "string" ? passkeyName.trim() : "",
			})

			if (dialogID) {
				const dialog = document.getElementById(dialogID)
				if (dialog && dialog.open) {
					dialog.close()
				}
			}

			dispatchAccountUpdated()

			if (typeof finishPayload.recoveryCodesToken !== "string" || finishPayload.recoveryCodesToken.length === 0) {
				throw new Error("The backup codes could not be loaded. Please regenerate them.")
			}
			await window.simpledmsShowRecoveryCodes(finishPayload.recoveryCodesToken)
			window.simpledmsShowClientSnackbar("Passkey registered.")
		} catch (err) {
			if (err instanceof Error) {
				throw err
			}
			throw new Error("Passkey registration failed.")
		}
	}

		document.body.addEventListener("passkeyRecoveryCodesGenerated", async function(e) {
		const detail = e && e.detail ? e.detail : {}
		const value = detail && typeof detail.value === "object" && detail.value !== null ? detail.value : detail
		const recoveryCodesToken = typeof value.recoveryCodesToken === "string" ? value.recoveryCodesToken : ""
		const noBackupCodesReturnedMessage = passkeyRuntimeMessage(
			"noBackupCodesReturned",
			"No backup codes were returned.",
		)
		const backupCodesRegeneratedMessage = passkeyRuntimeMessage(
			"backupCodesRegenerated",
			"The backup codes were regenerated.",
		)
		const couldNotRegenerateBackupCodesMessage = passkeyRuntimeMessage(
			"couldNotRegenerateBackupCodes",
			"Could not regenerate backup codes.",
		)

		if (recoveryCodesToken.length === 0) {
			window.simpledmsShowClientSnackbar(noBackupCodesReturnedMessage)
			return
		}

		try {
			await window.simpledmsShowRecoveryCodes(recoveryCodesToken)
			window.simpledmsShowClientSnackbar(backupCodesRegeneratedMessage)
		} catch (err) {
			window.simpledmsShowClientSnackbar(
				err instanceof Error ? err.message : couldNotRegenerateBackupCodesMessage,
			)
		}
	})
})()
