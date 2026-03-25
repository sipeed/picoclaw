import HTMLElement from '../html-element/HTMLElement.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import NodeList from '../node/NodeList.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTMLProgressElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLProgressElement
 */
export default class HTMLProgressElement extends HTMLElement {
	/**
	 * Returns max.
	 *
	 * @returns Max.
	 */
	public get max(): number {
		if (!this.hasAttribute('max')) {
			return 1;
		}
		const parsedValue = parseFloat(this.getAttribute('max') || '');
		if (isNaN(parsedValue) || parsedValue < 0) {
			return 1;
		}
		return parsedValue;
	}

	/**
	 * Sets max.
	 *
	 * @param max Max.
	 */
	public set max(max: number) {
		max = typeof max !== 'number' ? Number(max) : max;
		if (isNaN(max)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'max' property on 'HTMLProgressElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('max', max < 0 ? '1' : String(max));
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): number {
		if (!this.hasAttribute('value')) {
			return 0;
		}
		const parsedValue = parseFloat(this.getAttribute('value') || '');
		if (isNaN(parsedValue) || parsedValue < 0) {
			return 0;
		}
		return parsedValue;
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: number) {
		value = typeof value !== 'number' ? Number(value) : value;
		if (isNaN(value)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'value' property on 'HTMLProgressElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('value', value < 0 ? '0' : String(value));
	}

	/**
	 * Returns position.
	 *
	 * @returns Position.
	 */
	public get position(): number {
		// If the progress bar is an indeterminate progress bar, it should return -1.
		// It is considered indeterminate if the value attribute is not set.
		// @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLProgressElement/position#value
		if (!this.hasAttribute('value')) {
			return -1;
		}
		return this.value / this.max;
	}

	/**
	 * Returns the associated label elements.
	 *
	 * @returns Label elements.
	 */
	public get labels(): NodeList<HTMLLabelElement> {
		return HTMLLabelElementUtility.getAssociatedLabelElements(this);
	}
}
