import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import HTMLInputElementSelectionDirectionEnum from '../html-input-element/HTMLInputElementSelectionDirectionEnum.js';
import HTMLInputElementSelectionModeEnum from '../html-input-element/HTMLInputElementSelectionModeEnum.js';
import ValidityState from '../../validity-state/ValidityState.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import NodeList from '../node/NodeList.js';

/**
 * HTML Text Area Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTextAreaElement.
 */
export default class HTMLTextAreaElement extends HTMLElement {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLTextAreaElement;
	public readonly type = 'textarea';

	// Events
	public oninput: ((event: Event) => void) | null = null;
	public onselectionchange: ((event: Event) => void) | null = null;

	// Internal properties
	public [PropertySymbol.validationMessage] = '';
	public [PropertySymbol.validity] = new ValidityState(this);
	public [PropertySymbol.value] = null;
	public [PropertySymbol.textAreaNode] = this;
	public [PropertySymbol.formNode]: HTMLFormElement | null = null;

	// Private properties
	#selectionStart = null;
	#selectionEnd = null;
	#selectionDirection = HTMLInputElementSelectionDirectionEnum.none;

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
	 * Returns the default value.
	 *
	 * @returns Default value.
	 */
	public get defaultValue(): string {
		return this.textContent;
	}

	/**
	 * Sets the default value.
	 *
	 * @param defaultValue Default value.
	 */
	public set defaultValue(defaultValue: string) {
		this.textContent = defaultValue;
	}

	/**
	 * Returns minlength.
	 *
	 * @returns Min length.
	 */
	public get minLength(): number {
		const minLength = this.getAttribute('minlength');
		if (minLength !== null) {
			return parseInt(minLength);
		}
		return -1;
	}

	/**
	 * Sets minlength.
	 *
	 * @param minLength Min length.
	 */
	public set minLength(minlength: number) {
		this.setAttribute('minlength', String(minlength));
	}

	/**
	 * Returns maxlength.
	 *
	 * @returns Max length.
	 */
	public get maxLength(): number {
		const maxLength = this.getAttribute('maxlength');
		if (maxLength !== null) {
			return parseInt(maxLength);
		}
		return -1;
	}

	/**
	 * Sets maxlength.
	 *
	 * @param maxlength Max length.
	 */
	public set maxLength(maxLength: number) {
		this.setAttribute('maxlength', String(maxLength));
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
	 * Returns placeholder.
	 *
	 * @returns Placeholder.
	 */
	public get placeholder(): string {
		return this.getAttribute('placeholder') || '';
	}

	/**
	 * Sets placeholder.
	 *
	 * @param placeholder Placeholder.
	 */
	public set placeholder(placeholder: string) {
		this.setAttribute('placeholder', placeholder);
	}

	/**
	 * Returns inputmode.
	 *
	 * @returns Inputmode.
	 */
	public get inputMode(): string {
		return this.getAttribute('inputmode') || '';
	}

	/**
	 * Sets inputmode.
	 *
	 * @param inputmode Inputmode.
	 */
	public set inputMode(inputmode: string) {
		this.setAttribute('inputmode', inputmode);
	}

	/**
	 * Returns cols.
	 *
	 * @returns Cols.
	 */
	public get cols(): string {
		return this.getAttribute('cols') || '';
	}

	/**
	 * Sets cols.
	 *
	 * @param cols Cols.
	 */
	public set cols(cols: string) {
		this.setAttribute('cols', cols);
	}

	/**
	 * Returns rows.
	 *
	 * @returns Rows.
	 */
	public get rows(): string {
		return this.getAttribute('rows') || '';
	}

	/**
	 * Sets rows.
	 *
	 * @param rows Rows.
	 */
	public set rows(rows: string) {
		this.setAttribute('rows', rows);
	}

	/**
	 * Returns autocomplete.
	 *
	 * @returns Autocomplete.
	 */
	public get autocomplete(): string {
		return this.getAttribute('autocomplete') || '';
	}

	/**
	 * Sets autocomplete.
	 *
	 * @param autocomplete Autocomplete.
	 */
	public set autocomplete(autocomplete: string) {
		this.setAttribute('autocomplete', autocomplete);
	}

	/**
	 * Returns readOnly.
	 *
	 * @returns ReadOnly.
	 */
	public get readOnly(): boolean {
		return this.getAttribute('readonly') !== null;
	}

	/**
	 * Sets readOnly.
	 *
	 * @param readOnly ReadOnly.
	 */
	public set readOnly(readOnly: boolean) {
		if (!readOnly) {
			this.removeAttribute('readonly');
		} else {
			this.setAttribute('readonly', '');
		}
	}

	/**
	 * Returns disabled.
	 *
	 * @returns Disabled.
	 */
	public get disabled(): boolean {
		return this.getAttribute('disabled') !== null;
	}

	/**
	 * Sets disabled.
	 *
	 * @param disabled Disabled.
	 */
	public set disabled(disabled: boolean) {
		if (!disabled) {
			this.removeAttribute('disabled');
		} else {
			this.setAttribute('disabled', '');
		}
	}

	/**
	 * Returns autofocus.
	 *
	 * @returns Autofocus.
	 */
	public get autofocus(): boolean {
		return this.getAttribute('autofocus') !== null;
	}

	/**
	 * Sets autofocus.
	 *
	 * @param autofocus Autofocus.
	 */
	public set autofocus(autofocus: boolean) {
		if (!autofocus) {
			this.removeAttribute('autofocus');
		} else {
			this.setAttribute('autofocus', '');
		}
	}

	/**
	 * Returns required.
	 *
	 * @returns Required.
	 */
	public get required(): boolean {
		return this.getAttribute('required') !== null;
	}

	/**
	 * Sets required.
	 *
	 * @param required Required.
	 */
	public set required(required: boolean) {
		if (!required) {
			this.removeAttribute('required');
		} else {
			this.setAttribute('required', '');
		}
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		if (this[PropertySymbol.value] === null) {
			return this.textContent;
		}

		return this[PropertySymbol.value];
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		const oldValue = this[PropertySymbol.value];
		this[PropertySymbol.value] = value;

		if (oldValue !== this[PropertySymbol.value]) {
			this.#selectionStart = this[PropertySymbol.value].length;
			this.#selectionEnd = this[PropertySymbol.value].length;
			this.#selectionDirection = HTMLInputElementSelectionDirectionEnum.none;
		}
	}

	/**
	 * Returns selection start.
	 *
	 * @returns Selection start.
	 */
	public get selectionStart(): number {
		if (this.#selectionStart === null) {
			return this.value.length;
		}

		return this.#selectionStart;
	}

	/**
	 * Sets selection start.
	 *
	 * @param start Start.
	 */
	public set selectionStart(start: number) {
		this.setSelectionRange(start, Math.max(start, this.selectionEnd), this.#selectionDirection);
	}

	/**
	 * Returns selection end.
	 *
	 * @returns Selection end.
	 */
	public get selectionEnd(): number {
		if (this.#selectionEnd === null) {
			return this.value.length;
		}

		return this.#selectionEnd;
	}

	/**
	 * Sets selection end.
	 *
	 * @param end End.
	 */
	public set selectionEnd(end: number) {
		this.setSelectionRange(this.selectionStart, end, this.#selectionDirection);
	}

	/**
	 * Returns selection direction.
	 *
	 * @returns Selection direction.
	 */
	public get selectionDirection(): string {
		return this.#selectionDirection;
	}

	/**
	 * Sets selection direction.
	 *
	 * @param direction Direction.
	 */
	public set selectionDirection(direction: string) {
		this.setSelectionRange(this.selectionStart, this.selectionEnd, direction);
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
	 * Returns text length.
	 *
	 * @param Text Length.
	 */
	public get textLength(): number {
		return this.value.length;
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
	 * Selects the text.
	 */
	public select(): void {
		this.#selectionStart = 0;
		this.#selectionEnd = this.value.length;
		this.#selectionDirection = HTMLInputElementSelectionDirectionEnum.none;

		this.dispatchEvent(new Event('select', { bubbles: true, cancelable: false }));
	}

	/**
	 * Set selection range.
	 *
	 * @param start Start.
	 * @param end End.
	 * @param [direction="none"] Direction.
	 */
	public setSelectionRange(start: number, end: number, direction = 'none'): void {
		this.#selectionEnd = Math.min(end, this.value.length);
		this.#selectionStart = Math.min(start, this.selectionEnd);
		this.#selectionDirection =
			direction === HTMLInputElementSelectionDirectionEnum.forward ||
			direction === HTMLInputElementSelectionDirectionEnum.backward
				? direction
				: HTMLInputElementSelectionDirectionEnum.none;
		this.dispatchEvent(new Event('select', { bubbles: true, cancelable: false }));
	}

	/**
	 * Set range text.
	 *
	 * @param replacement Replacement.
	 * @param [start] Start.
	 * @param [end] End.
	 * @param [direction] Direction.
	 * @param selectionMode
	 */
	public setRangeText(
		replacement: string,
		start: number = null,
		end: number = null,
		selectionMode = HTMLInputElementSelectionModeEnum.preserve
	): void {
		if (start === null) {
			start = this.#selectionStart;
		}
		if (end === null) {
			end = this.#selectionEnd;
		}

		if (start > end) {
			throw new this[PropertySymbol.window].DOMException(
				'The index is not in the allowed range.',
				DOMExceptionNameEnum.invalidStateError
			);
		}

		start = Math.min(start, this.value.length);
		end = Math.min(end, this.value.length);

		const val = this.value;
		let selectionStart = this.#selectionStart;
		let selectionEnd = this.#selectionEnd;

		this.value = val.slice(0, start) + replacement + val.slice(end);

		const newEnd = start + this.value.length;

		switch (selectionMode) {
			case HTMLInputElementSelectionModeEnum.select:
				this.setSelectionRange(start, newEnd);
				break;
			case HTMLInputElementSelectionModeEnum.start:
				this.setSelectionRange(start, start);
				break;
			case HTMLInputElementSelectionModeEnum.end:
				this.setSelectionRange(newEnd, newEnd);
				break;
			default:
				const delta = replacement.length - (end - start);

				if (selectionStart > end) {
					selectionStart += delta;
				} else if (selectionStart > start) {
					selectionStart = start;
				}

				if (selectionEnd > end) {
					selectionEnd += delta;
				} else if (selectionEnd > start) {
					selectionEnd = newEnd;
				}

				this.setSelectionRange(selectionStart, selectionEnd);
				break;
		}
	}

	/**
	 * Sets validation message.
	 *
	 * @param message Message.
	 */
	public setCustomValidity(message: string): void {
		this[PropertySymbol.validationMessage] = String(message);
	}

	/**
	 * Checks validity.
	 *
	 * @returns "true" if the field is valid.
	 */
	public checkValidity(): boolean {
		const valid = this.disabled || this.readOnly || this[PropertySymbol.validity].valid;
		if (!valid) {
			this.dispatchEvent(new Event('invalid', { bubbles: true, cancelable: true }));
		}
		return valid;
	}

	/**
	 * Reports validity.
	 *
	 * @returns "true" if the field is valid.
	 */
	public reportValidity(): boolean {
		return this.checkValidity();
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLTextAreaElement {
		const clone = <HTMLTextAreaElement>super[PropertySymbol.cloneNode](deep);

		clone[PropertySymbol.value] = this[PropertySymbol.value];
		clone.#selectionStart = this.#selectionStart;
		clone.#selectionEnd = this.#selectionEnd;
		clone.#selectionDirection = this.#selectionDirection;

		return clone;
	}

	/**
	 * Resets selection.
	 */
	public [PropertySymbol.resetSelection](): void {
		if (this[PropertySymbol.value] === null) {
			this.#selectionStart = null;
			this.#selectionEnd = null;
			this.#selectionDirection = HTMLInputElementSelectionDirectionEnum.none;
		}
	}
}
