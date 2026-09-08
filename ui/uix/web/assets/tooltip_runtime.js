// Plain-text adaptation of Erdikon's shared tooltip popover.
(() => {
	const tooltip = document.createElement('div');
	tooltip.id = 'app-tooltip';
	tooltip.className = 'js-tooltip-popover';
	tooltip.setAttribute('popover', 'manual');
	tooltip.setAttribute('role', 'tooltip');
	document.body.append(tooltip);

	let activeTrigger = null;
	let describedElement = null;
	let showTimer = null;
	let hideTimer = null;
	let isKeyboardNavigation = false;

	const clearTimers = () => {
		clearTimeout(showTimer);
		clearTimeout(hideTimer);
	};
	const findTrigger = target => {
		const trigger = target instanceof Element ? target.closest('[data-tooltip]') : null;
		return trigger?.dataset.tooltip.trim() ? trigger : null;
	};
	const hide = () => {
		clearTimers();
		if (tooltip.matches(':popover-open')) tooltip.hidePopover();
		if (describedElement) {
			const ids = (describedElement.getAttribute('aria-describedby') || '')
				.split(/\s+/).filter(id => id && id !== tooltip.id);
			if (ids.length) describedElement.setAttribute('aria-describedby', ids.join(' '));
			else describedElement.removeAttribute('aria-describedby');
		}
		activeTrigger = null;
		describedElement = null;
		tooltip.textContent = '';
		document.body.append(tooltip);
	};
	const show = trigger => {
		if (!trigger.isConnected || !trigger.getClientRects().length) return;
		hide();
		activeTrigger = trigger;
		describedElement = trigger.contains(document.activeElement) ? document.activeElement : trigger;
		const ids = (describedElement.getAttribute('aria-describedby') || '').split(/\s+/).filter(Boolean);
		describedElement.setAttribute('aria-describedby', [...ids, tooltip.id].join(' '));
		tooltip.textContent = trigger.dataset.tooltip;
		// A tooltip outside a modal dialog would be inert, even in the top layer.
		(trigger.closest('dialog') || document.body).append(tooltip);
		tooltip.style.left = '0px';
		tooltip.style.top = '0px';
		tooltip.showPopover();
		const rect = trigger.getBoundingClientRect();
		const tip = tooltip.getBoundingClientRect();
		const clamp = (value, max) => Math.max(8, Math.min(value, max));
		let top = rect.top - tip.height - 8;
		if (top < 8) top = rect.bottom + 8;
		tooltip.style.left = `${clamp(rect.left + (rect.width - tip.width) / 2,
			window.innerWidth - tip.width - 8)}px`;
		tooltip.style.top = `${clamp(top, window.innerHeight - tip.height - 8)}px`;
	};
	const scheduleShow = (trigger, delay) => {
		clearTimers();
		if (activeTrigger !== trigger) showTimer = setTimeout(() => show(trigger), delay);
	};
	const scheduleHide = () => {
		clearTimers();
		hideTimer = setTimeout(hide, 120);
	};

	document.addEventListener('pointerover', event => {
		if (event.pointerType === 'touch') return;
		if (tooltip.contains(event.target)) {
			clearTimers();
			return;
		}
		const trigger = findTrigger(event.target);
		if (trigger) scheduleShow(trigger, 450);
	});
	document.addEventListener('pointerout', event => {
		const trigger = findTrigger(event.target);
		if (trigger?.contains(event.relatedTarget) || tooltip.contains(event.relatedTarget)) return;
		// Keep keyboard descriptions visible until focus leaves the control.
		if (isKeyboardNavigation && trigger?.contains(document.activeElement)) return;
		if (trigger || tooltip.contains(event.target)) scheduleHide();
	});
	document.addEventListener('focusin', event => {
		const trigger = findTrigger(event.target);
		if (trigger && isKeyboardNavigation) scheduleShow(trigger, 0);
	});
	document.addEventListener('focusout', event => {
		if (findTrigger(event.target)) scheduleHide();
	});
	document.addEventListener('pointerdown', () => {
		isKeyboardNavigation = false;
		hide();
	}, true);
	document.addEventListener('keydown', event => {
		if (event.key === 'Tab' || event.key.startsWith('Arrow')) isKeyboardNavigation = true;
		if (event.key === 'Escape') {
			if (activeTrigger) {
				event.preventDefault();
				event.stopImmediatePropagation();
			}
			hide();
		}
	}, true);
	document.addEventListener('click', hide, true);
	document.addEventListener('contextmenu', hide, true);
	document.addEventListener('scroll', event => {
		if (!tooltip.contains(event.target)) hide();
	}, true);
	document.addEventListener('beforetoggle', event => {
		if (event.target !== tooltip) hide();
	}, true);
	window.addEventListener('resize', hide);
	window.addEventListener('blur', hide);
	document.addEventListener('htmx:beforeRequest', hide);
	document.addEventListener('htmx:beforeSwap', hide);
})();
