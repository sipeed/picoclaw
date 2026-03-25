import * as PropertySymbol from '../PropertySymbol.js';
import HTMLInputElement from '../nodes/html-input-element/HTMLInputElement.js';
import HTMLTextAreaElement from '../nodes/html-text-area-element/HTMLTextAreaElement.js';
import HTMLObjectElement from '../nodes/html-object-element/HTMLObjectElement.js';
import HTMLOutputElement from '../nodes/html-output-element/HTMLOutputElement.js';
const EMAIL_REGEXP = /^([^\x00-\x20\x22\x28\x29\x2c\x2e\x3a-\x3c\x3e\x40\x5b-\x5d\x7f-\xff]+|\x22([^\x0d\x22\x5c\x80-\xff]|\x5c[\x00-\x7f])*\x22)(\x2e([^\x00-\x20\x22\x28\x29\x2c\x2e\x3a-\x3c\x3e\x40\x5b-\x5d\x7f-\xff]+|\x22([^\x0d\x22\x5c\x80-\xff]|\x5c[\x00-\x7f])*\x22))*\x40([^\x00-\x20\x22\x28\x29\x2c\x2e\x3a-\x3c\x3e\x40\x5b-\x5d\x7f-\xff]+|\x5b([^\x0d\x5b-\x5d\x80-\xff]|\x5c[\x00-\x7f])*\x5d)(\x2e([^\x00-\x20\x22\x28\x29\x2c\x2e\x3a-\x3c\x3e\x40\x5b-\x5d\x7f-\xff]+|\x5b([^\x0d\x5b-\x5d\x80-\xff]|\x5c[\x00-\x7f])*\x5d))*$/;
const URL_REGEXP = /^(?:(?:https?|HTTPS?|ftp|FTP):\/\/)(?:\S+(?::\S*)?@)?(?:(?!(?:10|127)(?:\.\d{1,3}){3})(?!(?:169\.254|192\.168)(?:\.\d{1,3}){2})(?!172\.(?:1[6-9]|2\d|3[0-1])(?:\.\d{1,3}){2})(?:[1-9]\d?|1\d\d|2[01]\d|22[0-3])(?:\.(?:1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.(?:[1-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(?:(?:[a-zA-Z\u00a1-\uffff0-9]-*)*[a-zA-Z\u00a1-\uffff0-9]+)(?:\.(?:[a-zA-Z\u00a1-\uffff0-9]-*)*[a-zA-Z\u00a1-\uffff0-9]+)*)(?::\d{2,5})?(?:[\/?#]\S*)?$/;
/**
 * Input validity state.
 *
 * Based on:
 * https://github.com/cferdinandi/validate/blob/master/src/js/_validityState.polyfill.js
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/ValidityState
 */
export default class ValidityState {
    element;
    /**
     * Constructor.
     *
     * @param element Input element.
     */
    constructor(element) {
        this.element = element;
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get badInput() {
        return (this.element instanceof HTMLInputElement &&
            (this.element.type === 'number' || this.element.type === 'range') &&
            this.element.value.length > 0 &&
            !/^[-+]?(?:\d+|\d*[.,]\d+)$/.test(this.element.value));
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get customError() {
        return this.element[PropertySymbol.validationMessage].length > 0;
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get patternMismatch() {
        return (this.element instanceof HTMLInputElement &&
            this.element.hasAttribute('pattern') &&
            this.element.value.length > 0 &&
            this.element.value.replace(new RegExp(this.element.getAttribute('pattern')), '').length > 0);
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get rangeOverflow() {
        return (this.element instanceof HTMLInputElement &&
            this.element.hasAttribute('max') &&
            (this.element.type === 'number' || this.element.type === 'range') &&
            this.element.value.length > 0 &&
            Number(this.element.value) > Number(this.element.getAttribute('max')));
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get rangeUnderflow() {
        return (this.element instanceof HTMLInputElement &&
            this.element.hasAttribute('min') &&
            (this.element.type === 'number' || this.element.type === 'range') &&
            this.element.value.length > 0 &&
            Number(this.element.value) < Number(this.element.getAttribute('min')));
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get stepMismatch() {
        return (this.element instanceof HTMLInputElement &&
            (this.element.type === 'number' || this.element.type === 'range') &&
            ((this.element.hasAttribute('step') &&
                this.element.getAttribute('step') !== 'any' &&
                Number(this.element.value) % Number(this.element.getAttribute('step')) !== 0) ||
                (!this.element.hasAttribute('step') && Number(this.element.value) % 1 !== 0)));
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get tooLong() {
        return ((this.element instanceof HTMLInputElement || this.element instanceof HTMLTextAreaElement) &&
            this.element.maxLength > 0 &&
            this.element.value.length > this.element.maxLength);
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get tooShort() {
        return ((this.element instanceof HTMLInputElement || this.element instanceof HTMLTextAreaElement) &&
            this.element.minLength > 0 &&
            this.element.value.length > 0 &&
            this.element.value.length < this.element.minLength);
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get typeMismatch() {
        return (this.element instanceof HTMLInputElement &&
            this.element.value.length > 0 &&
            ((this.element.type === 'email' && !EMAIL_REGEXP.test(this.element.value)) ||
                (this.element.type === 'url' && !URL_REGEXP.test(this.element.value))));
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get valueMissing() {
        if (!this.element.required ||
            this.element instanceof HTMLObjectElement ||
            this.element instanceof HTMLOutputElement) {
            return false;
        }
        if (this.element instanceof HTMLInputElement) {
            if (this.element.type === 'checkbox') {
                return !this.element.checked;
            }
            else if (this.element.type === 'radio') {
                if (this.element.checked) {
                    return false;
                }
                if (!this.element.name) {
                    return true;
                }
                const root = this.element[PropertySymbol.formNode] ||
                    this.element.getRootNode();
                return !root || !root.querySelector(`input[name="${this.element.name}"]:checked`);
            }
        }
        return this.element.value.length === 0;
    }
    /**
     * Returns validity.
     *
     * @returns "true" if valid.
     */
    get valid() {
        return (!this.badInput &&
            !this.customError &&
            !this.patternMismatch &&
            !this.rangeOverflow &&
            !this.rangeUnderflow &&
            !this.stepMismatch &&
            !this.tooLong &&
            !this.tooShort &&
            !this.typeMismatch &&
            !this.valueMissing);
    }
}
//# sourceMappingURL=ValidityState.js.map