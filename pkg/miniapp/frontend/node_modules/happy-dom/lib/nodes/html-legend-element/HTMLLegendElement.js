import HTMLElement from '../html-element/HTMLElement.js';
import HTMLFieldSetElement from '../html-field-set-element/HTMLFieldSetElement.js';
/**
 * HTMLLegendElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLLegendElement
 */
export default class HTMLLegendElement extends HTMLElement {
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        let parent = this;
        while (parent) {
            if (parent instanceof HTMLFieldSetElement) {
                return parent.form;
            }
            parent = parent.parentNode;
        }
        return null;
    }
}
//# sourceMappingURL=HTMLLegendElement.js.map