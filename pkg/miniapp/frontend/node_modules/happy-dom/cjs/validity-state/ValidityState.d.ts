import HTMLButtonElement from '../nodes/html-button-element/HTMLButtonElement.cjs';
import HTMLInputElement from '../nodes/html-input-element/HTMLInputElement.cjs';
import HTMLSelectElement from '../nodes/html-select-element/HTMLSelectElement.cjs';
import HTMLTextAreaElement from '../nodes/html-text-area-element/HTMLTextAreaElement.cjs';
import HTMLObjectElement from '../nodes/html-object-element/HTMLObjectElement.cjs';
import HTMLOutputElement from '../nodes/html-output-element/HTMLOutputElement.cjs';
/**
 * Input validity state.
 *
 * Based on:
 * https://github.com/cferdinandi/validate/blob/master/src/js/_validityState.polyfill.js
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/ValidityState
 */
export default class ValidityState {
    private element;
    /**
     * Constructor.
     *
     * @param element Input element.
     */
    constructor(element: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement | HTMLObjectElement | HTMLOutputElement);
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get badInput(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get customError(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get patternMismatch(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get rangeOverflow(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get rangeUnderflow(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get stepMismatch(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get tooLong(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get tooShort(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get typeMismatch(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get valueMissing(): boolean;
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get valid(): boolean;
}
//# sourceMappingURL=ValidityState.d.ts.map