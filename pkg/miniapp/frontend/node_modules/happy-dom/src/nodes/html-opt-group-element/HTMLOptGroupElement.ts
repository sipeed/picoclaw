import HTMLElement from '../html-element/HTMLElement.js';

/**
 * HTML Opt Group Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptGroupElement
 */
export default class HTMLOptGroupElement extends HTMLElement {
	/**
	 * Returns label.
	 *
	 * @returns Label.
	 */
	public get label(): string {
		return this.getAttribute('label') || '';
	}

	/**
	 * Sets label.
	 *
	 * @param label Label.
	 */
	public set label(label: string) {
		if (!label) {
			this.removeAttribute('label');
		} else {
			this.setAttribute('label', label);
		}
	}

	/**
	 * Returns disabled.
	 *
	 * @returns Disabled.
	 */
	public get disabled(): boolean {
		return this.getAttribute('disabled') !== null;
	}

	/**
	 * Sets disabled.
	 *
	 * @param disabled Disabled.
	 */
	public set disabled(disabled: boolean) {
		if (!disabled) {
			this.removeAttribute('disabled');
		} else {
			this.setAttribute('disabled', '');
		}
	}
}
