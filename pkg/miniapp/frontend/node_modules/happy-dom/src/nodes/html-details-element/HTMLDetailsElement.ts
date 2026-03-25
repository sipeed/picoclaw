import Event from '../../event/Event.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Attr from '../attr/Attr.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import MouseEvent from '../../event/events/MouseEvent.js';

/**
 * HTMLDetailsElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDetailsElement
 */
export default class HTMLDetailsElement extends HTMLElement {
	// Events
	public ontoggle: ((event: Event) => void) | null = null;

	/**
	 * Returns the open attribute.
	 */
	public get open(): boolean {
		return this.getAttribute('open') !== null;
	}

	/**
	 * Sets the open attribute.
	 *
	 * @param open New value.
	 */
	public set open(open: boolean) {
		if (open) {
			this.setAttribute('open', '');
		} else {
			this.removeAttribute('open');
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onSetAttribute](
		attribute: Attr,
		replacedAttribute: Attr | null
	): void {
		super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
		if (attribute[PropertySymbol.name] === 'open') {
			if (attribute[PropertySymbol.value] !== replacedAttribute?.[PropertySymbol.value]) {
				this.dispatchEvent(new Event('toggle'));
			}
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onRemoveAttribute](removedAttribute: Attr): void {
		super[PropertySymbol.onRemoveAttribute](removedAttribute);
		if (removedAttribute && removedAttribute[PropertySymbol.name] === 'open') {
			this.dispatchEvent(new Event('toggle'));
		}
	}

	/**
	 * @override
	 */
	public override dispatchEvent(event: Event): boolean {
		const returnValue = super.dispatchEvent(event);

		if (
			!event[PropertySymbol.defaultPrevented] &&
			event[PropertySymbol.target]?.[PropertySymbol.localName] === 'summary' &&
			event.type === 'click' &&
			event.eventPhase === EventPhaseEnum.bubbling &&
			event instanceof MouseEvent
		) {
			this.open = !this.open;
		}

		return returnValue;
	}
}
