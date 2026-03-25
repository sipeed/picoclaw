import BrowserWindow from '../../window/BrowserWindow.js';
import Document from '../document/Document.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import ValidityState from '../../validity-state/ValidityState.js';

/**
 * HTMLObjectElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLObjectElement
 */
export default class HTMLObjectElement extends HTMLElement {
	public [PropertySymbol.formNode]: HTMLFormElement | null = null;
	public [PropertySymbol.validationMessage] = '';
	public [PropertySymbol.validity] = new ValidityState(this);

	/**
	 * Returns the content document.
	 *
	 * @returns Document
	 */
	public get contentDocument(): Document | null {
		return null;
	}

	/**
	 * Returns the content window.
	 *
	 * @returns Window
	 */
	public get contentWindow(): BrowserWindow | null {
		return null;
	}

	/**
	 * Returns source.
	 *
	 * @returns Source.
	 */
	public get data(): string {
		if (!this.hasAttribute('data')) {
			return '';
		}

		try {
			return new URL(this.getAttribute('data'), this[PropertySymbol.ownerDocument].location.href)
				.href;
		} catch (e) {
			return this.getAttribute('data');
		}
	}

	/**
	 * Sets source.
	 *
	 * @param data Source.
	 */
	public set data(data: string) {
		this.setAttribute('data', data);
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
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): string {
		return this.getAttribute('height') || '';
	}

	/**
	 * Sets height.
	 *
	 * @param height Height.
	 */
	public set height(height: string) {
		this.setAttribute('height', height);
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): string {
		return this.getAttribute('width') || '';
	}

	/**
	 * Sets width.
	 *
	 * @param width Width.
	 */
	public set width(width: string) {
		this.setAttribute('width', width);
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
	 * Returns type.
	 *
	 * @returns Type.
	 */
	public get type(): string {
		return this.getAttribute('type') || '';
	}

	/**
	 * Sets type.
	 *
	 * @param type Type.
	 */
	public set type(type: string) {
		this.setAttribute('type', type);
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
