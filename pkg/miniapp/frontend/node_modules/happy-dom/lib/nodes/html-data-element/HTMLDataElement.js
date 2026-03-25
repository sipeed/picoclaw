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
    get value() {
        return this.getAttribute('value') || '';
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this.setAttribute('value', value);
    }
}
//# sourceMappingURL=HTMLDataElement.js.map