import HTMLElement from '../html-element/HTMLElement.js';

/**
 * HTMLTimeElement.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTimeElement
 */
export default class HTMLTimeElement extends HTMLElement {
	/**
	 * Returns dateTime.
	 *
	 * @returns dateTime.
	 */
	public get dateTime(): string {
		return this.getAttribute('dateTime') || '';
	}

	/**
	 * Sets dateTime.
	 *
	 * @param dateTime dateTime.
	 */
	public set dateTime(dateTime: string) {
		this.setAttribute('dateTime', dateTime);
	}
}
