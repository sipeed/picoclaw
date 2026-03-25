import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTMLDataElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDataElement
 */
export default class HTMLDataElement extends HTMLElement {
	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		return this.getAttribute('value') || '';
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		this.setAttribute('value', value);
	}
}
