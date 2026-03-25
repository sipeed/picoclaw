import HTMLElement from '../html-element/HTMLElement.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
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
    get max() {
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
    set max(max) {
        max = typeof max !== 'number' ? Number(max) : max;
        if (isNaN(max)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'max' property on 'HTMLProgressElement': The provided double value is non-finite.");
        }
        this.setAttribute('max', max < 0 ? '1' : String(max));
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
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
    set value(value) {
        value = typeof value !== 'number' ? Number(value) : value;
        if (isNaN(value)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'value' property on 'HTMLProgressElement': The provided double value is non-finite.");
        }
        this.setAttribute('value', value < 0 ? '0' : String(value));
    }
    /**
     * Returns position.
     *
     * @returns Position.
     */
    get position() {
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
    get labels() {
        return HTMLLabelElementUtility.getAssociatedLabelElements(this);
    }
}
//# sourceMappingURL=HTMLProgressElement.js.map