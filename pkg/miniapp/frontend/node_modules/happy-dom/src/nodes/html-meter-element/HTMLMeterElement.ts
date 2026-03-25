import HTMLElement from '../html-element/HTMLElement.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import NodeList from '../node/NodeList.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTMLMeterElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLMeterElement
 */
export default class HTMLMeterElement extends HTMLElement {
	/**
	 * Returns high.
	 *
	 * @returns High.
	 */
	public get high(): number {
		if (!this.hasAttribute('high')) {
			return 1;
		}
		const parsedValue = parseFloat(this.getAttribute('high') || '');
		if (isNaN(parsedValue) || parsedValue > 1) {
			return 1;
		}
		return parsedValue < 0 ? 0 : parsedValue;
	}

	/**
	 * Sets high.
	 *
	 * @param high High.
	 */
	public set high(high: number) {
		high = typeof high !== 'number' ? Number(high) : high;
		if (isNaN(high)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'high' property on 'HTMLMeterElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('high', String(high));
	}

	/**
	 * Returns low.
	 *
	 * @returns Low.
	 */
	public get low(): number {
		if (!this.hasAttribute('low')) {
			return 0;
		}
		const parsedValue = parseFloat(this.getAttribute('low') || '');
		if (isNaN(parsedValue) || parsedValue < 0) {
			return 0;
		}
		return parsedValue > 1 ? 1 : parsedValue;
	}

	/**
	 * Sets low.
	 *
	 * @param low Low.
	 */
	public set low(low: number) {
		low = typeof low !== 'number' ? Number(low) : low;
		if (isNaN(low)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'low' property on 'HTMLMeterElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('low', String(low));
	}

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
		if (isNaN(parsedValue) || parsedValue > 1) {
			return 1;
		}
		return parsedValue < 0 ? 0 : parsedValue;
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
				"Failed to set the 'max' property on 'HTMLMeterElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('max', String(max));
	}

	/**
	 * Returns min.
	 *
	 * @returns Min.
	 */
	public get min(): number {
		if (!this.hasAttribute('min')) {
			return 0;
		}
		const parsedValue = parseFloat(this.getAttribute('min') || '');
		if (isNaN(parsedValue) || parsedValue < 0) {
			return 0;
		}
		return parsedValue > 1 ? 1 : parsedValue;
	}

	/**
	 * Sets min.
	 *
	 * @param min Min.
	 */
	public set min(min: number) {
		min = typeof min !== 'number' ? Number(min) : min;
		if (isNaN(min)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'min' property on 'HTMLMeterElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('min', String(min));
	}

	/**
	 * Returns optimum.
	 *
	 * @returns Optimum.
	 */
	public get optimum(): number {
		if (!this.hasAttribute('optimum')) {
			return 0.5;
		}
		const parsedValue = parseFloat(this.getAttribute('optimum') || '');
		if (isNaN(parsedValue)) {
			return 0.5;
		}
		if (parsedValue < 0) {
			return 0;
		}
		return parsedValue > 1 ? 1 : parsedValue;
	}

	/**
	 * Sets optimum.
	 *
	 * @param optimum Optimum.
	 */
	public set optimum(optimum: number) {
		optimum = typeof optimum !== 'number' ? Number(optimum) : optimum;
		if (isNaN(optimum)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'optimum' property on 'HTMLMeterElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('optimum', String(optimum));
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
		return parsedValue > 1 ? 1 : parsedValue;
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
				"Failed to set the 'value' property on 'HTMLMeterElement': The provided double value is non-finite."
			);
		}

		this.setAttribute('value', String(value));
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
