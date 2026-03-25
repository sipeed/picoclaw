import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTML Base Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/HTML/Element/base.
 */
export default class HTMLBaseElement extends HTMLElement {
	public declare cloneNode: (deep?: boolean) => HTMLBaseElement;

	/**
	 * Returns href.
	 *
	 * @returns Href.
	 */
	public get href(): string {
		if (!this.hasAttribute('href')) {
			return this[PropertySymbol.ownerDocument].location.href;
		}

		try {
			return new URL(this.getAttribute('href'), this[PropertySymbol.ownerDocument].location.href)
				.href;
		} catch (e) {
			return this.getAttribute('href');
		}
	}

	/**
	 * Sets href.
	 *
	 * @param href Href.
	 */
	public set href(href: string) {
		this.setAttribute('href', href);
	}

	/**
	 * Returns target.
	 *
	 * @returns Target.
	 */
	public get target(): string {
		return this.getAttribute('target') || '';
	}

	/**
	 * Sets target.
	 *
	 * @param target Target.
	 */
	public set target(target: string) {
		this.setAttribute('target', target);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLBaseElement {
		return <HTMLBaseElement>super[PropertySymbol.cloneNode](deep);
	}
}
