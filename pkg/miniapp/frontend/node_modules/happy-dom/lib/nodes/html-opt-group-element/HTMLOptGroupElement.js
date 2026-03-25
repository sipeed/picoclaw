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
    get label() {
        return this.getAttribute('label') || '';
    }
    /**
     * Sets label.
     *
     * @param label Label.
     */
    set label(label) {
        if (!label) {
            this.removeAttribute('label');
        }
        else {
            this.setAttribute('label', label);
        }
    }
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled() {
        return this.getAttribute('disabled') !== null;
    }
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled) {
        if (!disabled) {
            this.removeAttribute('disabled');
        }
        else {
            this.setAttribute('disabled', '');
        }
    }
}
//# sourceMappingURL=HTMLOptGroupElement.js.map