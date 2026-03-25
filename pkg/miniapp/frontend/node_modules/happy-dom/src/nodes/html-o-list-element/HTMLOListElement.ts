import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTMLOListElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLOListElement
 */
export default class HTMLOListElement extends HTMLElement {
	/**
	 * Returns reversed.
	 *
	 * @returns Reversed.
	 */
	public get reversed(): boolean {
		return this.getAttribute('reversed') !== null;
	}

	/**
	 * Sets reversed.
	 *
	 * @param reversed Reversed.
	 */
	public set reversed(reversed: boolean) {
		if (!reversed) {
			this.removeAttribute('reversed');
		} else {
			this.setAttribute('reversed', '');
		}
	}

	/**
	 * Returns start.
	 *
	 * @returns Start.
	 */
	public get start(): number {
		if (!this.hasAttribute('start')) {
			return 1;
		}

		const parsedValue = Number(this.getAttribute('start') || '');

		if (isNaN(parsedValue)) {
			return 1;
		}

		return parsedValue;
	}

	/**
	 * Sets start.
	 *
	 * @param start Start.
	 */
	public set start(start: number) {
		if (typeof start !== 'number') {
			start = Number(start);
		}

		if (isNaN(start)) {
			start = 0;
		}

		this.setAttribute('start', String(start));
	}

	/**
	 * Returns type.
	 *
	 * @returns Type.
	 */
	public get type(): string {
		return this.getAttribute('type') || '';
	}

	/**
	 * Sets type.
	 *
	 * @param type Type.
	 */
	public set type(type: string) {
		this.setAttribute('type', type);
	}
}
