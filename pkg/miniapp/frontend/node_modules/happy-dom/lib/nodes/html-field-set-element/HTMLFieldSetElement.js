import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLCollection from '../element/HTMLCollection.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
/**
 * HTMLFieldSetElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFieldSetElement
 */
export default class HTMLFieldSetElement extends HTMLElement {
    // Internal properties
    [PropertySymbol.elements] = null;
    [PropertySymbol.formNode] = null;
    /**
     * Returns elements.
     *
     * @returns Elements.
     */
    get elements() {
        if (!this[PropertySymbol.elements]) {
            this[PropertySymbol.elements] = new HTMLCollection(PropertySymbol.illegalConstructor, () => (QuerySelector.querySelectorAll(this, 'input,button,textarea,select')[PropertySymbol.items]));
        }
        return this[PropertySymbol.elements];
    }
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        if (this[PropertySymbol.formNode]) {
            return this[PropertySymbol.formNode];
        }
        const id = this.getAttribute('form');
        if (!id || !this[PropertySymbol.isConnected]) {
            return null;
        }
        return this[PropertySymbol.ownerDocument].getElementById(id);
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this.getAttribute('name') || '';
    }
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name) {
        this.setAttribute('name', name);
    }
    /**
     * Returns type "fieldset".
     *
     * @returns Type.
     */
    get type() {
        return 'fieldset';
    }
    /**
     * Returns empty string as fieldset never candidates for constraint validation.
     */
    get validationMessage() {
        return '';
    }
    /**
     * Returns will validate state.
     *
     * Always returns false as fieldset never candidates for constraint validation.
     *
     * @returns Will validate state.
     */
    get willValidate() {
        return false;
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
    /**
     * Checks validity.
     *
     * Always returns true as fieldset never candidates for constraint validation.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity() {
        return true;
    }
    /**
     * Reports validity.
     *
     * Always returns true as fieldset never candidates for constraint validation.
     *
     * @returns Validity.
     */
    reportValidity() {
        return true;
    }
    /**
     * Sets validation message.
     *
     * Does nothing as fieldset never candidates for constraint validation.
     *
     * @param _message Message.
     */
    setCustomValidity(_message) {
        // Do nothing as fieldset never candidates for constraint validation.
    }
}
//# sourceMappingURL=HTMLFieldSetElement.js.map