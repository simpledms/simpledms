(function() {
	function snackbarTarget() {
		var activeElement = document.activeElement;
		if (activeElement && typeof activeElement.closest === "function") {
			var activeDialog = activeElement.closest("dialog[open].js-dialog");
			if (activeDialog) {
				return activeDialog;
			}
		}

		var openDialogs = document.querySelectorAll("dialog[open].js-dialog:not(:has(dialog[open].js-dialog))");
		if (openDialogs.length > 0) {
			return openDialogs[openDialogs.length - 1];
		}

		return document.body;
	}

	window.simpledmsMountSnackbar = function(snackbarRoot, autoDismissTimeoutInMs) {
		if (!snackbarRoot) {
			return function() {};
		}

		document.dispatchEvent(new CustomEvent("closeAllSnackbars", {}));

		var observer = null;
		var timeoutID = null;
		var isClosed = false;
		var dismissTimeoutInMs = typeof autoDismissTimeoutInMs === "number" ? autoDismissTimeoutInMs : 5000;

		var closeSnackbar = function() {
			if (isClosed) {
				return;
			}

			isClosed = true;
			document.removeEventListener("closeAllSnackbars", closeSnackbar);
			if (timeoutID !== null) {
				clearTimeout(timeoutID);
			}
			if (observer !== null) {
				observer.disconnect();
			}
			snackbarRoot.remove();
		};

		document.addEventListener("closeAllSnackbars", closeSnackbar);

		var snackbar = snackbarRoot.querySelector(".js-snackbar");
		if (snackbar) {
			snackbar.addEventListener("click", closeSnackbar);
		}

		observer = new IntersectionObserver(function(entries) {
			entries.forEach(function(entry) {
				if (!entry.isIntersecting) {
					var target = snackbarTarget();
					if (target) {
						target.appendChild(snackbarRoot);
					}
				}
			});
		});
		observer.observe(snackbarRoot);

		timeoutID = setTimeout(closeSnackbar, dismissTimeoutInMs);

		return closeSnackbar;
	};

	// TODO not a nice solution, refactor, used only for passkeys integration
	window.simpledmsShowClientSnackbar = function(message, options) {
		if (!message) {
			return;
		}

		var opts = options || {};
		var wrapper = document.createElement("div");
		wrapper.id = opts.id || ("simpledms-client-snackbar-" + Date.now().toString(36) + Math.random().toString(36).slice(2, 8));

		var snackbar = document.createElement("div");
		snackbar.setAttribute("popover", "manual");
		snackbar.className = "js-snackbar bottom-24 flex top-auto group mx-auto left-4 right-4 max-w-md bg-inverse-surface drop-shadow rounded-xs p-0 hover:cursor-pointer";

		var inner = document.createElement("div");
		inner.className = "state-layer-inverse-primary flex min-h-12 max-h-[4.25rem] items-center justify-center gap-x-4 px-4";

		var text = document.createElement("div");
		text.className = "text-inverse-on-surface body-md group-hover:text-inverse-primary";
		text.textContent = message;

		inner.appendChild(text);
		snackbar.appendChild(inner);
		wrapper.appendChild(snackbar);

		var target = snackbarTarget();
		if (!target) {
			target = document.body;
		}
		target.appendChild(wrapper);

		return window.simpledmsMountSnackbar(wrapper, opts.autoDismissTimeoutInMs || 5000);
	};
})();
