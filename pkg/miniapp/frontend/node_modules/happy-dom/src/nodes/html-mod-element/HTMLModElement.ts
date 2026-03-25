import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTMLModElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLModElement
 */
export default class HTMLModElement extends HTMLElement {
	/**
	 * Returns source.
	 *
	 * @returns Source.
	 */
	public get cite(): string {
		if (!this.hasAttribute('cite')) {
			return '';
		}

		try {
			return new URL(this.getAttribute('cite'), this[PropertySymbol.ownerDocument].location.href)
				.href;
		} catch (e) {
			return this.getAttribute('cite');
		}
	}

	/**
	 * Sets source.
	 *
	 * @param cite Source.
	 */
	public set cite(cite: string) {
		this.setAttribute('cite', cite);
	}

	/**
	 * Returns date time.
	 *
	 * @returns Date time.
	 */
	public get dateTime(): string {
		return this.getAttribute('datetime') || '';
	}

	/**
	 * Sets date time.
	 *
	 * @param dateTime Date time.
	 */
	public set dateTime(dateTime: string) {
		this.setAttribute('datetime', dateTime);
	}
}
