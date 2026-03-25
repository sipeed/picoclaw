import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ValidityState from '../../validity-state/ValidityState.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
/**
 * HTMLOutputElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLOutputElement
 */
export default class HTMLOutputElement extends HTMLElement {
    [PropertySymbol.formNode] = null;
    [PropertySymbol.validationMessage] = '';
    [PropertySymbol.validity] = new ValidityState(this);
    [PropertySymbol.defaultValue] = '';
    /**
     * Returns default value.
     *
     * @returns Default value.
     */
    get defaultValue() {
        return this[PropertySymbol.defaultValue];
    }
    /**
     * Sets default value.
     *
     * @param defaultValue Default value.
     */
    set defaultValue(defaultValue) {
        this[PropertySymbol.defaultValue] = defaultValue;
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
     * Returns a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @returns ID of the labeled control.
     */
    get htmlFor() {
        return this.getAttribute('for') || '';
    }
    /**
     * Sets a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @param htmlFor ID of the labeled control.
     */
    set htmlFor(htmlFor) {
        this.setAttribute('for', htmlFor);
    }
    /**
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels() {
        return HTMLLabelElementUtility.getAssociatedLabelElements(this);
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
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this.textContent || '';
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this.textContent = value;
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return 'output';
    }
    /**
     * Returns validation message.
     *
     * @returns Validation message.
     */
    get validationMessage() {
        return this[PropertySymbol.validationMessage];
    }
    /**
     * Returns validity.
     *
     * @returns Validity.
     */
    get validity() {
        return this[PropertySymbol.validity];
    }
    /**
     * Returns "true" if it will validate.
     *
     * @returns "true" if it will validate.
     */
    get willValidate() {
        return false;
    }
    /**
     * Checks validity.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity() {
        return true;
    }
    /**
     * Reports validity.
     *
     * @returns Validity.
     */
    reportValidity() {
        return this.checkValidity();
    }
    /**
     * Sets validation message.
     *
     * @param message Message.
     */
    setCustomValidity(message) {
        this[PropertySymbol.validationMessage] = String(message);
    }
}
//# sourceMappingURL=HTMLOutputElement.js.map