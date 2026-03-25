import HTMLElement from '../html-element/HTMLElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLFormElement from '../html-form-element/HTMLFormElement.cjs';
import ValidityState from '../../validity-state/ValidityState.cjs';
import NodeList from '../node/NodeList.cjs';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.cjs';
/**
 * HTMLOutputElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLOutputElement
 */
export default class HTMLOutputElement extends HTMLElement {
    [PropertySymbol.formNode]: HTMLFormElement | null;
    [PropertySymbol.validationMessage]: string;
    [PropertySymbol.validity]: ValidityState;
    [PropertySymbol.defaultValue]: string;
    /**
     * Returns default value.
     *
     * @returns Default value.
     */
    get defaultValue(): string;
    /**
     * Sets default value.
     *
     * @param defaultValue Default value.
     */
    set defaultValue(defaultValue: string);
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form(): HTMLFormElement;
    /**
     * Returns a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @returns ID of the labeled control.
     */
    get htmlFor(): string;
    /**
     * Sets a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @param htmlFor ID of the labeled control.
     */
    set htmlFor(htmlFor: string);
    /**
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels(): NodeList<HTMLLabelElement>;
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
     * Returns value.
     *
     * @returns Value.
     */
    get value(): string;
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value: string);
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): string;
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
//# sourceMappingURL=HTMLOutputElement.d.ts.map