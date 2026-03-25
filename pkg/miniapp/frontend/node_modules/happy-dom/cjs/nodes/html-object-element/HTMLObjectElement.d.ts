import BrowserWindow from '../../window/BrowserWindow.cjs';
import Document from '../document/Document.cjs';
import HTMLElement from '../html-element/HTMLElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLFormElement from '../html-form-element/HTMLFormElement.cjs';
import ValidityState from '../../validity-state/ValidityState.cjs';
/**
 * HTMLObjectElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLObjectElement
 */
export default class HTMLObjectElement extends HTMLElement {
    [PropertySymbol.formNode]: HTMLFormElement | null;
    [PropertySymbol.validationMessage]: string;
    [PropertySymbol.validity]: ValidityState;
    /**
     * Returns the content document.
     *
     * @returns Document
     */
    get contentDocument(): Document | null;
    /**
     * Returns the content window.
     *
     * @returns Window
     */
    get contentWindow(): BrowserWindow | null;
    /**
     * Returns source.
     *
     * @returns Source.
     */
    get data(): string;
    /**
     * Sets source.
     *
     * @param data Source.
     */
    set data(data: string);
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form(): HTMLFormElement;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): string;
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height: string);
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): string;
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width: string);
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
     * Returns type.
     *
     * @returns Type.
     */
    get type(): string;
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type: string);
    /**
     * Returns validation message.
     *
     * @returns Validation message.
     */
    get validationMessage(): string;
    /**
     * Returns validity.
     *
     * @returns Validity.
     */
    get validity(): ValidityState;
    /**
     * Returns "true" if it will validate.
     *
     * @returns "true" if it will validate.
     */
    get willValidate(): boolean;
    /**
     * Checks validity.
     *
     * @returns "true" if the field is valid.
     */
    checkValidity(): boolean;
    /**
     * Reports validity.
     *
     * @returns Validity.
     */
    reportValidity(): boolean;
    /**
     * Sets validation message.
     *
     * @param message Message.
     */
    setCustomValidity(message: string): void;
}
//# sourceMappingURL=HTMLObjectElement.d.ts.map