import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import ValidityState from '../../validity-state/ValidityState.js';
import NodeList from '../node/NodeList.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';

/**
 * HTMLOutputElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLOutputElement
 */
export default class HTMLOutputElement extends HTMLElement {
	public [PropertySymbol.formNode]: HTMLFormElement | null = null;
	public [PropertySymbol.validationMessage] = '';
	public [PropertySymbol.validity] = new ValidityState(this);
	public [PropertySymbol.defaultValue] = '';

	/**
	 * Returns default value.
	 *
	 * @returns Default value.
	 */
	public get defaultValue(): string {
		return this[PropertySymbol.defaultValue];
	}

	/**
	 * Sets default value.
	 *
	 * @param defaultValue Default value.
	 */
	public set defaultValue(defaultValue: string) {
		this[PropertySymbol.defaultValue] = defaultValue;
	}

	/**
	 * Returns the parent form element.
	 *
	 * @returns Form.
	 */
	public get form(): HTMLFormElement {
		if (this[PropertySymbol.formNode]) {
			return this[PropertySymbol.formNode];
		}
		const id = this.getAttribute('form');
		if (!id || !this[PropertySymbol.isConnected]) {
			return null;
		}
		return <HTMLFormElement>this[PropertySymbol.ownerDocument].getElementById(id);
	}

	/**
	 * Returns a string containing the ID of the labeled control. This reflects the "for" attribute.
	 *
	 * @returns ID of the labeled control.
	 */
	public get htmlFor(): string {
		return this.getAttribute('for') || '';
	}

	/**
	 * Sets a string containing the ID of the labeled control. This reflects the "for" attribute.
	 *
	 * @param htmlFor ID of the labeled control.
	 */
	public set htmlFor(htmlFor: string) {
		this.setAttribute('for', htmlFor);
	}

	/**
	 * Returns the associated label elements.
	 *
	 * @returns Label elements.
	 */
	public get labels(): NodeList<HTMLLabelElement> {
		return HTMLLabelElementUtility.getAssociatedLabelElements(this);
	}

	/**
	 * Returns name.
	 *
	 * @returns Name.
	 */
	public get name(): string {
		return this.getAttribute('name') || '';
	}

	/**
	 * Sets name.
	 *
	 * @param name Name.
	 */
	public set name(name: string) {
		this.setAttribute('name', name);
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		return this.textContent || '';
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		this.textContent = value;
	}

	/**
	 * Returns type.
	 *
	 * @returns Type.
	 */
	public get type(): string {
		return 'output';
	}

	/**
	 * Returns validation message.
	 *
	 * @returns Validation message.
	 */
	public get validationMessage(): string {
		return this[PropertySymbol.validationMessage];
	}

	/**
	 * Returns validity.
	 *
	 * @returns Validity.
	 */
	public get validity(): ValidityState {
		return this[PropertySymbol.validity];
	}

	/**
	 * Returns "true" if it will validate.
	 *
	 * @returns "true" if it will validate.
	 */
	public get willValidate(): boolean {
		return false;
	}

	/**
	 * Checks validity.
	 *
	 * @returns "true" if the field is valid.
	 */
	public checkValidity(): boolean {
		return true;
	}

	/**
	 * Reports validity.
	 *
	 * @returns Validity.
	 */
	public reportValidity(): boolean {
		return this.checkValidity();
	}

	/**
	 * Sets validation message.
	 *
	 * @param message Message.
	 */
	public setCustomValidity(message: string): void {
		this[PropertySymbol.validationMessage] = String(message);
	}
}
