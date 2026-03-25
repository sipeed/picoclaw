import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLCollection from '../element/HTMLCollection.js';
import HTMLInputElement from '../html-input-element/HTMLInputElement.js';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.js';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.js';
import HTMLButtonElement from '../html-button-element/HTMLButtonElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
type THTMLFieldSetElement = HTMLInputElement | HTMLButtonElement | HTMLTextAreaElement | HTMLSelectElement;
/**
 * HTMLFieldSetElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFieldSetElement
 */
export default class HTMLFieldSetElement extends HTMLElement {
    cloneNode: (deep?: boolean) => HTMLFieldSetElement;
    [PropertySymbol.elements]: HTMLCollection<THTMLFieldSetElement> | null;
    [PropertySymbol.formNode]: HTMLFormElement | null;
    /**
     * Returns elements.
     *
     * @returns Elements.
     */
    get elements(): HTMLCollection<THTMLFieldSetElement>;
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form(): HTMLFormElement;
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name(): string;
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name: string);
    /**
     * Returns type "fieldset".
     *
     * @returns Type.
     */
    get type(): string;
    /**
     * Returns empty string as fieldset never candidates for constraint validation.
     */
    get validationMessage(): string;
    /**
     * Returns will validate state.
     *
     * Always returns false as fieldset never candidates for constraint validation.
     *
     * @returns Will validate state.
     */
    get willValidate(): boolean;
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled(): boolean;
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled: boolean);
    /**
     * Checks validity.
     *
     * Always returns true as fieldset never candidates for constraint validation.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity(): boolean;
    /**
     * Reports validity.
     *
     * Always returns true as fieldset never candidates for constraint validation.
     *
     * @returns Validity.
     */
    reportValidity(): boolean;
    /**
     * Sets validation message.
     *
     * Does nothing as fieldset never candidates for constraint validation.
     *
     * @param _message Message.
     */
    setCustomValidity(_message: string): void;
}
export {};
//# sourceMappingURL=HTMLFieldSetElement.d.ts.map