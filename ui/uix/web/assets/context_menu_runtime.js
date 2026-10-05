// Context menus of list items, table rows, and cards. Owners are marked with
// data-context-menu; document-level listeners replace a script per row, which was costly to
// send and execute for long lists.
(() => {
	// necessary for nested list items: the summary is the owner's area, otherwise a right click
	// on a child would also open the parent's context menu
	const ownerArea = owner => owner.querySelector('summary') ?? owner;
	const contextMenuOf = owner => ownerArea(owner).querySelector('menu');
	const parentOwner = owner => owner.parentElement?.closest('[data-context-menu]');

	const ownerForTarget = target => {
		let owner = target instanceof Element ? target.closest('[data-context-menu]') : null;
		while (owner) {
			if (ownerArea(owner).contains(target) && contextMenuOf(owner)) return owner;
			owner = parentOwner(owner);
		}
		return null;
	};
	const ownerOfMenu = menu => {
		let owner = menu.closest('[data-context-menu]');
		while (owner) {
			if (contextMenuOf(owner) === menu) return owner;
			owner = parentOwner(owner);
		}
		return null;
	};

	// Pointer coordinates are viewport-relative, not relative to the Actions anchor.
	// Restore the anchor styles on close for subsequent button/keyboard opening.
	const anchoredStyles = new WeakMap();
	const openListeners = new WeakMap();

	const listenersFor = (menu, owner) => {
		// duplicate in icon_button.gohtml
		const closePopover = e => {
			if (!menu.contains(e.target)) {
				// in case of right click, just close and don't open
				// native context menu or next popover;
				// in case of left click, don't execute action
				// TODO not sure if best solution for right click, might
				//		be handy to open next popover menu? browser default
				//		for native context menus is current behavior
				e.preventDefault();
				e.stopImmediatePropagation(); // necessary for left click
			} else if (e.target.closest('a[href]') === null) {
				// added on 18 April 2025 to prevent closing a wrapping <details> element when
				// selecting an item from the context menu; was necessary in manage tags for
				// editing tag groups; not sure if it has side effects
				e.preventDefault();
			}

			// also on click inside the menu, otherwise menu stays open in
			// the background and first click in popover does nothing but
			// closing the menu in the background
			menu.hidePopover();
		};
		const closeOnEscape = e => {
			if (e.key !== 'Escape') return;
			e.preventDefault();
			e.stopPropagation();
			menu.hidePopover();
			ownerArea(owner).querySelector(`[popovertarget="${menu.id}"]`)?.focus();
		};
		return { closePopover, closeOnEscape };
	};

	// beforetoggle doesn't bubble, but capture listeners on the document receive it. It is
	// synchronous: Escape must work immediately after opening.
	document.addEventListener('beforetoggle', e => {
		const menu = e.target;
		if (!(menu instanceof HTMLMenuElement)) return;
		const owner = ownerOfMenu(menu);
		if (!owner) return;

		const isOpening = e.newState === 'open';
		ownerArea(owner).querySelector(`[popovertarget="${menu.id}"]`)
			?.setAttribute('aria-expanded', String(isOpening));

		if (isOpening) {
			const listeners = listenersFor(menu, owner);
			openListeners.set(menu, listeners);
			// true means that it is executed in early `capture` phase
			// necessary for left click to prevent execution of target action,
			// not sure for right click
			document.addEventListener('click', listeners.closePopover, true);
			document.addEventListener('contextmenu', listeners.closePopover, true);
			document.addEventListener('keydown', listeners.closeOnEscape, true);
			return;
		}

		const listeners = openListeners.get(menu);
		if (listeners) {
			document.removeEventListener('click', listeners.closePopover, true);
			document.removeEventListener('contextmenu', listeners.closePopover, true);
			document.removeEventListener('keydown', listeners.closeOnEscape, true);
			openListeners.delete(menu);
		}
		if (anchoredStyles.has(menu)) {
			menu.style.cssText = anchoredStyles.get(menu);
			anchoredStyles.delete(menu);
		}
	}, true);

	document.addEventListener('contextmenu', e => {
		// implemented by default in firefox
		if (e.getModifierState('Shift')) return;
		const owner = ownerForTarget(e.target);
		if (!owner) return;
		const menu = contextMenuOf(owner);

		e.preventDefault();
		anchoredStyles.set(menu, menu.style.cssText);
		menu.style.positionArea = 'none';
		menu.style.inset = 'auto';
		// TODO not good if no space on right side, TODO autoposition
		menu.style.top = e.clientY + 'px';
		menu.style.left = e.clientX + 'px';
		menu.showPopover();
	});
})();
