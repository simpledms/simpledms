(function () {
	if (window.__simpleDMSNavigationRailRuntimeLoaded) {
		return;
	}
	window.__simpleDMSNavigationRailRuntimeLoaded = true;

	const defaultStorageKey = 'simpledms.navigationRail.expanded';
	const expandedMediaQuery = window.matchMedia('(min-width: 600px)');
	const collapsedRailWidth = '80px';
	const expandedRailWidth = '360px';

	const isCompact = () => !expandedMediaQuery.matches;

	const storageGet = (key) => {
		try {
			return window.localStorage.getItem(key);
		} catch (_) {
			return null;
		}
	};

	const storageSet = (key, value) => {
		try {
			window.localStorage.setItem(key, value);
		} catch (_) {
			// Local UI preference only. Ignore unavailable storage.
		}
	};

	const railStorageKey = (rail) => rail.dataset.navigationRailStorageKey || defaultStorageKey;
	const groupStorageKey = (rail, group) => `${railStorageKey(rail)}.group.${group.dataset.navigationRailGroupKey}`;

	const setCurrentRailWidth = (expanded) => {
		document.documentElement.style.setProperty(
			'--simpledms-navigation-rail-width',
			isCompact() ? '0px' : expanded ? expandedRailWidth : collapsedRailWidth,
		);
	};

	const forEachRailToggle = (rail, callback) => {
		document.querySelectorAll('.js-navigation-rail-toggle').forEach((toggle) => {
			const toggleRail = toggle.closest('.js-navigation-rail');
			const targetID = toggle.dataset.navigationRailTarget || toggle.getAttribute('aria-controls');
			if (toggleRail === rail || targetID === rail.id) {
				callback(toggle);
			}
		});
	};

	const railForToggle = (toggle) => {
		const rail = toggle.closest('.js-navigation-rail');
		if (rail) {
			return rail;
		}
		const targetID = toggle.dataset.navigationRailTarget || toggle.getAttribute('aria-controls');
		if (!targetID) {
			return null;
		}
		return document.getElementById(targetID);
	};

	const setExpanded = (rail, expanded, persist = true) => {
		if (isCompact()) {
			persist = false;
		}
		rail.classList.toggle('navigation-rail-expanded', expanded);
		rail.dataset.expanded = expanded ? 'true' : 'false';
		rail.toggleAttribute('data-modal-open', isCompact() && expanded);
		setCurrentRailWidth(expanded);

		forEachRailToggle(rail, (toggle) => {
			toggle.setAttribute('aria-expanded', expanded ? 'true' : 'false');
			toggle.querySelectorAll('.js-navigation-rail-toggle-icon').forEach((icon) => {
				icon.textContent = expanded ? 'menu_open' : 'menu';
			});
		});
		rail.querySelectorAll('.js-navigation-rail-toggle-icon').forEach((icon) => {
			icon.textContent = expanded ? 'menu_open' : 'menu';
		});

		if (persist) {
			storageSet(railStorageKey(rail), expanded ? 'true' : 'false');
		}
	};

	const setGroupCollapsed = (group, collapsed, persist = true) => {
		group.dataset.collapsed = collapsed ? 'true' : 'false';
		group.querySelectorAll(':scope > .js-navigation-rail-group-toggle').forEach((toggle) => {
			toggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
		});
		group.querySelectorAll(':scope > .js-navigation-rail-group-toggle .js-navigation-rail-group-icon').forEach((icon) => {
			icon.textContent = collapsed ? 'keyboard_arrow_down' : 'keyboard_arrow_up';
		});

		const rail = group.closest('.js-navigation-rail');
		if (persist && rail && group.dataset.navigationRailGroupKey) {
			storageSet(groupStorageKey(rail, group), collapsed ? 'true' : 'false');
		}
	};

	const initGroup = (rail, group) => {
		let collapsed = group.dataset.collapsed === 'true';
		if (group.dataset.navigationRailGroupKey) {
			const stored = storageGet(groupStorageKey(rail, group));
			if (stored === 'true' || stored === 'false') {
				collapsed = stored === 'true';
			}
		}
		setGroupCollapsed(group, collapsed, false);
	};

	const initRail = (rail) => {
		const expanded = !isCompact() && storageGet(railStorageKey(rail)) === 'true';
		setExpanded(rail, expanded, false);
		rail.querySelectorAll('.js-navigation-rail-group').forEach((group) => initGroup(rail, group));
	};

	const initRails = (root = document) => {
		let initializedRail = false;
		if (root.matches && root.matches('.js-navigation-rail')) {
			initRail(root);
			initializedRail = true;
		}
		if (root.querySelectorAll) {
			root.querySelectorAll('.js-navigation-rail').forEach((rail) => {
				initRail(rail);
				initializedRail = true;
			});
		}
		if (!initializedRail && root.querySelector && root.querySelector('.js-navigation-rail-toggle')) {
			document.querySelectorAll('.js-navigation-rail').forEach((rail) => {
				setExpanded(rail, rail.classList.contains('navigation-rail-expanded'), false);
			});
		}
		if (!initializedRail) {
			const rail = document.querySelector('.js-navigation-rail');
			if (rail) {
				setExpanded(rail, rail.classList.contains('navigation-rail-expanded'), false);
				return;
			}
			setCurrentRailWidth(false);
		}
	};

	document.addEventListener('click', (event) => {
		const toggle = event.target.closest('.js-navigation-rail-toggle');
		if (toggle) {
			const rail = railForToggle(toggle);
			if (!rail) {
				return;
			}
			event.preventDefault();
			setExpanded(rail, !rail.classList.contains('navigation-rail-expanded'));
			return;
		}

		const scrim = event.target.closest('.js-navigation-rail-scrim');
		if (scrim) {
			const rail = scrim.closest('.js-navigation-rail');
			if (!rail) {
				return;
			}
			event.preventDefault();
			setExpanded(rail, false, false);
			return;
		}

		const groupToggle = event.target.closest('.js-navigation-rail-group-toggle');
		if (groupToggle) {
			const group = groupToggle.closest('.js-navigation-rail-group');
			if (!group) {
				return;
			}
			event.preventDefault();
			setGroupCollapsed(group, group.dataset.collapsed !== 'true');
			return;
		}

		const destination = event.target.closest('.js-navigation-rail a, .js-navigation-rail button');
		if (!destination || !isCompact()) {
			return;
		}
		const rail = destination.closest('.js-navigation-rail');
		if (rail && rail.classList.contains('navigation-rail-expanded')) {
			setExpanded(rail, false, false);
		}
	});

	document.addEventListener('keydown', (event) => {
		if (event.key !== 'Escape' || !isCompact()) {
			return;
		}
		document.querySelectorAll('.js-navigation-rail.navigation-rail-expanded').forEach((rail) => {
			setExpanded(rail, false, false);
		});
	});

	expandedMediaQuery.addEventListener('change', () => initRails());
	document.addEventListener('DOMContentLoaded', () => initRails());
	document.addEventListener('htmx:afterProcessNode', (event) => initRails(event.detail.elt));
	document.addEventListener('htmx:afterSwap', (event) => initRails(event.detail.elt));
	initRails();
})();
