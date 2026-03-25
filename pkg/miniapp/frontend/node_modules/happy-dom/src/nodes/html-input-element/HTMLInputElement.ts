import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ValidityState from '../../validity-state/ValidityState.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import Event from '../../event/Event.js';
import HTMLInputElementValueSanitizer from './HTMLInputElementValueSanitizer.js';
import HTMLInputElementSelectionModeEnum from './HTMLInputElementSelectionModeEnum.js';
import HTMLInputElementSelectionDirectionEnum from './HTMLInputElementSelectionDirectionEnum.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import HTMLInputElementValueStepping from './HTMLInputElementValueStepping.js';
import FileList from './FileList.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import HTMLInputElementDateUtility from './HTMLInputElementDateUtility.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import HTMLDataListElement from '../html-data-list-element/HTMLDataListElement.js';
import Document from '../document/Document.js';
import ShadowRoot from '../shadow-root/ShadowRoot.js';
import { URL } from 'url';
import MouseEvent from '../../event/events/MouseEvent.js';
import NodeList from '../node/NodeList.js';

/**
 * HTML Input Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLInputElement.
 *
 * Used as reference for some of the logic (like selection range):
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/nodes/nodes/HTMLInputElement-impl.js (MIT licensed).
 */
export default class HTMLInputElement extends HTMLElement {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLInputElement;

	// Events
	public oninput: ((event: Event) => void) | null = null;
	public oninvalid: ((event: Event) => void) | null = null;
	public onselectionchange: ((event: Event) => void) | null = null;

	public [PropertySymbol.value] = null;
	public [PropertySymbol.height] = 0;
	public [PropertySymbol.width] = 0;
	public [PropertySymbol.checked]: boolean | null = null;
	public [PropertySymbol.validationMessage] = '';
	public [PropertySymbol.validity] = new ValidityState(this);
	public [PropertySymbol.files]: FileList = new FileList();
	public [PropertySymbol.indeterminate]: boolean = false;
	public [PropertySymbol.formNode]: HTMLFormElement | null = null;
	public [PropertySymbol.popoverTargetElement]: HTMLElement | null = null;

	// Private properties
	#selectionStart: number = null;
	#selectionEnd: number = null;
	#selectionDirection: HTMLInputElementSelectionDirectionEnum =
		HTMLInputElementSelectionDirectionEnum.none;

	/**
	 * Returns default checked.
	 *
	 * @returns Default checked.
	 */
	public get defaultChecked(): boolean {
		return this.hasAttribute('checked');
	}

	/**
	 * Sets default checked.
	 *
	 * @param defaultChecked Default checked.
	 */
	public set defaultChecked(defaultChecked: boolean) {
		if (defaultChecked) {
			this.setAttribute('checked', '');
		} else {
			this.removeAttribute('checked');
		}
	}

	/**
	 * Returns files.
	 *
	 * @returns Files.
	 */
	public get files(): FileList {
		return this[PropertySymbol.files];
	}

	/**
	 * Sets files.
	 *
	 * @param files Files.
	 */
	public set files(files: FileList) {
		this[PropertySymbol.files] = files;
	}

	/**
	 * Returns form action.
	 *
	 * @returns Form action.
	 */
	public get formAction(): string {
		if (!this.hasAttribute('formaction')) {
			return this[PropertySymbol.ownerDocument].location.href;
		}

		try {
			return new URL(
				this.getAttribute('formaction'),
				this[PropertySymbol.ownerDocument].location.href
			).href;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets form action.
	 *
	 * @param formAction Form action.
	 */
	public set formAction(formAction: string) {
		this.setAttribute('formaction', formAction);
	}

	/**
	 * Returns form enctype.
	 *
	 * @returns Form enctype.
	 */
	public get formEnctype(): string {
		return this.getAttribute('formenctype') || '';
	}

	/**
	 * Sets form enctype.
	 *
	 * @param formEnctype Form enctype.
	 */
	public set formEnctype(formEnctype: string) {
		this.setAttribute('formenctype', formEnctype);
	}

	/**
	 * Returns form method.
	 *
	 * @returns Form method.
	 */
	public get formMethod(): string {
		return this.getAttribute('formmethod') || '';
	}

	/**
	 * Sets form method.
	 *
	 * @param formMethod Form method.
	 */
	public set formMethod(formMethod: string) {
		this.setAttribute('formmethod', formMethod);
	}

	/**
	 * Returns no validate.
	 *
	 * @returns No validate.
	 */
	public get formNoValidate(): boolean {
		return this.getAttribute('formnovalidate') !== null;
	}

	/**
	 * Sets no validate.
	 *
	 * @param formNoValidate No validate.
	 */
	public set formNoValidate(formNoValidate: boolean) {
		if (!formNoValidate) {
			this.removeAttribute('formnovalidate');
		} else {
			this.setAttribute('formnovalidate', '');
		}
	}

	/**
	 * Returns form target.
	 *
	 * @returns Form target.
	 */
	public get formTarget(): string {
		return this.getAttribute('formtarget') || '';
	}

	/**
	 * Sets form target.
	 *
	 * @param formTarget Form target.
	 */
	public set formTarget(formTarget: string) {
		this.setAttribute('formtarget', formTarget);
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
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): number {
		return this[PropertySymbol.height];
	}

	/**
	 * Sets height.
	 *
	 * @param height Height.
	 */
	public set height(height: number) {
		this[PropertySymbol.height] = height;
		this.setAttribute('height', String(height));
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): number {
		return this[PropertySymbol.width];
	}

	/**
	 * Sets width.
	 *
	 * @param width Width.
	 */
	public set width(width: number) {
		this[PropertySymbol.width] = width;
		this.setAttribute('width', String(width));
	}

	/**
	 * Returns size.
	 *
	 * @returns Size.
	 */
	public get size(): number {
		const size = this.getAttribute('size');
		if (size !== null) {
			return parseInt(size);
		}
		return 20;
	}

	/**
	 * Sets size.
	 *
	 * @param size Size.
	 */
	public set size(size: number) {
		this.setAttribute('size', String(size));
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
	 * Returns type.
	 *
	 * @returns Type. Defaults to "text".
	 */
	public get type(): string {
		return this.getAttribute('type') || 'text';
	}

	/**
	 * Sets type.
	 *
	 * @param type Type.
	 */
	public set type(type: string) {
		this.setAttribute('type', type.toLowerCase());
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
	 * Returns alt.
	 *
	 * @returns Alt.
	 */
	public get alt(): string {
		return this.getAttribute('alt') || '';
	}

	/**
	 * Sets alt.
	 *
	 * @param alt Alt.
	 */
	public set alt(alt: string) {
		this.setAttribute('alt', alt);
	}

	/**
	 * Returns min.
	 *
	 * @returns Min.
	 */
	public get min(): string {
		return this.getAttribute('min') || '';
	}

	/**
	 * Sets min.
	 *
	 * @param min Min.
	 */
	public set min(min: string) {
		this.setAttribute('min', min);
	}

	/**
	 * Returns max.
	 *
	 * @returns Max.
	 */
	public get max(): string {
		return this.getAttribute('max') || '';
	}

	/**
	 * Sets max.
	 *
	 * @param max Max.
	 */
	public set max(max: string) {
		this.setAttribute('max', max);
	}

	/**
	 * Returns pattern.
	 *
	 * @returns Pattern.
	 */
	public get pattern(): string {
		return this.getAttribute('pattern') || '';
	}

	/**
	 * Sets pattern.
	 *
	 * @param pattern Pattern.
	 */
	public set pattern(pattern: string) {
		this.setAttribute('pattern', pattern);
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
	 * Returns step.
	 *
	 * @returns Step.
	 */
	public get step(): string {
		return this.getAttribute('step') || '';
	}

	/**
	 * Sets step.
	 *
	 * @param step Step.
	 */
	public set step(step: string) {
		this.setAttribute('step', step);
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
	 * Returns accept.
	 *
	 * @returns Accept.
	 */
	public get accept(): string {
		return this.getAttribute('accept') || '';
	}

	/**
	 * Sets accept.
	 *
	 * @param accept Accept.
	 */
	public set accept(accept: string) {
		this.setAttribute('accept', accept);
	}

	/**
	 * Returns allowdirs.
	 *
	 * @returns Allowdirs.
	 */
	public get allowdirs(): string {
		return this.getAttribute('allowdirs') || '';
	}

	/**
	 * Sets allowdirs.
	 *
	 * @param allowdirs Allowdirs.
	 */
	public set allowdirs(allowdirs: string) {
		this.setAttribute('allowdirs', allowdirs);
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
	 * Returns src.
	 *
	 * @returns Src.
	 */
	public get src(): string {
		return this.getAttribute('src') || '';
	}

	/**
	 * Sets src.
	 *
	 * @param src Src.
	 */
	public set src(src: string) {
		this.setAttribute('src', src);
	}

	/**
	 * Returns defaultValue.
	 *
	 * @returns Defaultvalue.
	 */
	public get defaultValue(): string {
		return this.getAttribute('value') || '';
	}

	/**
	 * Sets defaultValue.
	 *
	 * @param defaultValue Defaultvalue.
	 */
	public set defaultValue(defaultValue: string) {
		this.setAttribute('value', defaultValue);
	}

	/**
	 * Returns read only.
	 *
	 * @returns Read only.
	 */
	public get readOnly(): boolean {
		return this.getAttribute('readonly') !== null;
	}

	/**
	 * Sets read only.
	 *
	 * @param readOnly Read only.
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
	 * Returns indeterminate.
	 *
	 * @returns Indeterminate.
	 */
	public get indeterminate(): boolean {
		return this[PropertySymbol.indeterminate];
	}

	/**
	 * Sets indeterminate.
	 *
	 * @param indeterminate Indeterminate.
	 */
	public set indeterminate(indeterminate: boolean) {
		this[PropertySymbol.indeterminate] = Boolean(indeterminate);
	}

	/**
	 * Returns multiple.
	 *
	 * @returns Multiple.
	 */
	public get multiple(): boolean {
		return this.getAttribute('multiple') !== null;
	}

	/**
	 * Sets multiple.
	 *
	 * @param multiple Multiple.
	 */
	public set multiple(multiple: boolean) {
		if (!multiple) {
			this.removeAttribute('multiple');
		} else {
			this.setAttribute('multiple', '');
		}
	}

	/**
	 * Returns checked.
	 *
	 * @returns Checked.
	 */
	public get checked(): boolean {
		if (this[PropertySymbol.checked] !== null) {
			return this[PropertySymbol.checked];
		}
		return this.getAttribute('checked') !== null;
	}

	/**
	 * Sets checked.
	 *
	 * @param checked Checked.
	 */
	public set checked(checked: boolean) {
		this.#setChecked(checked);
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		switch (this.type) {
			case 'hidden':
			case 'submit':
			case 'image':
			case 'reset':
			case 'button':
				return this.getAttribute('value') || '';
			case 'checkbox':
			case 'radio':
				const attritube = this.getAttribute('value');
				return attritube !== null ? attritube : 'on';
			case 'file':
				return this[PropertySymbol.files].length > 0
					? '/fake/path/' + this[PropertySymbol.files][0].name
					: '';
		}

		if (this[PropertySymbol.value] === null) {
			return this.getAttribute('value') || '';
		}

		return this[PropertySymbol.value];
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string | null) {
		// The value maybe not string, so we need to convert it to string
		value = value === null ? '' : String(value);
		switch (this.type) {
			case 'hidden':
			case 'submit':
			case 'image':
			case 'reset':
			case 'button':
			case 'checkbox':
			case 'radio':
				this.setAttribute('value', value);
				break;
			case 'file':
				if (value !== null && value !== '') {
					throw new this[PropertySymbol.window].DOMException(
						'Input elements of type "file" may only programmatically set the value to empty string.',
						DOMExceptionNameEnum.invalidStateError
					);
				}
				this[PropertySymbol.files] = new FileList();
				break;
			default:
				const oldValue = this.value;
				this[PropertySymbol.value] = HTMLInputElementValueSanitizer.sanitize(this, value);

				if (oldValue !== this[PropertySymbol.value]) {
					this.#selectionStart = this[PropertySymbol.value].length;
					this.#selectionEnd = this[PropertySymbol.value].length;
					this.#selectionDirection = HTMLInputElementSelectionDirectionEnum.none;
				}

				break;
		}
	}

	/**
	 * Returns selection start.
	 *
	 * @returns Selection start.
	 */
	public get selectionStart(): number {
		if (!this.#isSelectionSupported()) {
			return null;
		}

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
		if (!this.#isSelectionSupported()) {
			throw new this[PropertySymbol.window].DOMException(
				`The input element's type (${this.type}) does not support selection.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		this.setSelectionRange(start, Math.max(start, this.selectionEnd), this.#selectionDirection);
	}

	/**
	 * Returns selection end.
	 *
	 * @returns Selection end.
	 */
	public get selectionEnd(): number {
		if (!this.#isSelectionSupported()) {
			return null;
		}

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
		if (!this.#isSelectionSupported()) {
			throw new this[PropertySymbol.window].DOMException(
				`The input element's type (${this.type}) does not support selection.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		this.setSelectionRange(this.selectionStart, end, this.#selectionDirection);
	}

	/**
	 * Returns selection direction.
	 *
	 * @returns Selection direction.
	 */
	public get selectionDirection(): string {
		if (!this.#isSelectionSupported()) {
			return null;
		}

		return this.#selectionDirection;
	}

	/**
	 * Sets selection direction.
	 *
	 * @param direction Direction.
	 */
	public set selectionDirection(direction: string) {
		if (!this.#isSelectionSupported()) {
			throw new this[PropertySymbol.window].DOMException(
				`The input element's type (${this.type}) does not support selection.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		this.setSelectionRange(this.#selectionStart, this.#selectionEnd, direction);
	}

	/**
	 * Returns "true" if it will validate.
	 *
	 * @returns "true" if it will validate.
	 */
	public get willValidate(): boolean {
		return (
			this.type !== 'hidden' &&
			this.type !== 'reset' &&
			this.type !== 'button' &&
			!this.disabled &&
			!this['readOnly']
		);
	}

	/**
	 * Returns value as Date.
	 *
	 * @returns Date.
	 */
	public get valueAsDate(): Date {
		switch (this.type) {
			case 'date':
			case 'month':
				return isNaN(new Date(String(this.value)).getTime()) ? null : new Date(this.value);
			case 'week': {
				const d = HTMLInputElementDateUtility.isoWeekDate(this.value);
				return isNaN(d.getTime()) ? null : d;
			}
			case 'time': {
				const d = new Date(`1970-01-01T${this.value}Z`);
				return isNaN(d.getTime()) ? null : d;
			}
			default:
				return null;
		}
	}

	/**
	 * Sets value from a Date.
	 *
	 * @param value Date.
	 */
	public set valueAsDate(value: Date | null) {
		// Specs at https://html.spec.whatwg.org/multipage/input.html#dom-input-valueasdate
		if (!['date', 'month', 'time', 'week'].includes(this.type)) {
			throw new this[PropertySymbol.window].DOMException(
				"Failed to set the 'valueAsDate' property on 'HTMLInputElement': This input element does not support Date values.",
				DOMExceptionNameEnum.invalidStateError
			);
		}
		if (typeof value !== 'object') {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'valueAsDate' property on 'HTMLInputElement': Failed to convert value to 'object'."
			);
		} else if (value && !(value instanceof Date)) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to set the 'valueAsDate' property on 'HTMLInputElement': The provided value is not a Date."
			);
		} else if (value === null || isNaN(value.getTime())) {
			this.value = '';
			return;
		}
		switch (this.type) {
			case 'date':
				this.value = value.toISOString().split('T')[0];
				break;
			case 'month':
				this.value = value.toISOString().split('T')[0].slice(0, -3);
				break;
			case 'time':
				this.value = value.toISOString().split('T')[1].slice(0, 5);
				break;
			case 'week':
				this.value = HTMLInputElementDateUtility.dateIsoWeek(value);
				break;
		}
	}

	/**
	 * Returns value as number.
	 *
	 * @returns Number.
	 */
	public get valueAsNumber(): number {
		const value = this.value;
		if (!this.type.match(/^(range|number|date|datetime-local|month|time|week)$/) || !value) {
			return NaN;
		}
		switch (this.type) {
			case 'number':
				return parseFloat(value);
			case 'range': {
				const number = parseFloat(value);
				const min = parseFloat(this.min) || 0;
				const max = parseFloat(this.max) || 100;
				if (isNaN(number)) {
					return max < min ? min : (min + max) / 2;
				} else if (number < min) {
					return min;
				} else if (number > max) {
					return max;
				}
				return number;
			}
			case 'date':
				return new Date(value).getTime();
			case 'datetime-local':
				return new Date(value).getTime() - new Date(value).getTimezoneOffset() * 60000;
			case 'month':
				return (new Date(value).getUTCFullYear() - 1970) * 12 + new Date(value).getUTCMonth();
			case 'time':
				return (
					new Date('1970-01-01T' + value).getTime() - new Date('1970-01-01T00:00:00').getTime()
				);
			case 'week': {
				// https://html.spec.whatwg.org/multipage/input.html#week-state-(type=week)
				const match = value.match(/^(\d{4})-W(\d{2})$/);
				if (!match) {
					return NaN;
				}
				const d = new Date(Date.UTC(parseInt(match[1], 10), 0));
				const day = d.getUTCDay();
				const diff = ((day === 0 ? -6 : 1) - day) * 86400000 + parseInt(match[2], 10) * 604800000;
				return d.getTime() + diff;
			}
		}
	}

	/**
	 * Sets value from a number.
	 *
	 * @param value number.
	 */
	public set valueAsNumber(value: number) {
		// Specs at https://html.spec.whatwg.org/multipage/input.html
		switch (this.type) {
			case 'number':
			case 'range':
				// We Rely on HTMLInputElementValueSanitizer
				this.value = Number(value).toString();
				break;
			case 'date':
			case 'datetime-local': {
				const d = new Date(Number(value));
				if (isNaN(d.getTime())) {
					// Reset to default value
					this.value = '';
					break;
				}
				if (this.type == 'date') {
					this.value = d.toISOString().slice(0, 10);
				} else {
					this.value = d.toISOString().slice(0, -1);
				}
				break;
			}
			case 'month':
				if (!Number.isInteger(value) || value < 0) {
					this.value = '';
				} else {
					this.value = new Date(Date.UTC(1970, Number(value))).toISOString().slice(0, 7);
				}
				break;
			case 'time':
				if (!Number.isInteger(value) || value < 0) {
					this.value = '';
				} else {
					this.value = new Date(Number(value)).toISOString().slice(11, -1);
				}
				break;
			case 'week': {
				const d = new Date(Number(value));
				this.value = isNaN(d.getTime()) ? '' : HTMLInputElementDateUtility.dateIsoWeek(d);
				break;
			}
			default:
				throw new this[PropertySymbol.window].DOMException(
					"Failed to set the 'valueAsNumber' property on 'HTMLInputElement': This input element does not support Number values.",
					DOMExceptionNameEnum.invalidStateError
				);
		}
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
	 * Returns associated datalist element.
	 *
	 * @returns Data list element.
	 */
	public get list(): HTMLDataListElement | null {
		const id = this.getAttribute('list');
		if (!id) {
			return null;
		}
		const rootNode =
			<Document | ShadowRoot>this[PropertySymbol.rootNode] || this[PropertySymbol.ownerDocument];
		return <HTMLDataListElement | null>rootNode.querySelector(`datalist#${id}`);
	}

	/**
	 * Returns popover target element.
	 *
	 * @returns Popover target element.
	 */
	public get popoverTargetElement(): HTMLElement | null {
		return this[PropertySymbol.popoverTargetElement];
	}

	/**
	 * Sets popover target element.
	 *
	 * @param popoverTargetElement Popover target element.
	 */
	public set popoverTargetElement(popoverTargetElement: HTMLElement | null) {
		if (popoverTargetElement !== null && !(popoverTargetElement instanceof HTMLElement)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'popoverTargetElement' property on 'HTMLInputElement': Failed to convert value to 'Element'.`
			);
		}
		this[PropertySymbol.popoverTargetElement] = popoverTargetElement;
	}

	/**
	 * Returns popover target action.
	 *
	 * @returns Popover target action.
	 */
	public get popoverTargetAction(): string {
		const value = this.getAttribute('popovertargetaction');
		if (value === null || (value !== 'hide' && value !== 'show' && value !== 'toggle')) {
			return 'toggle';
		}
		return value;
	}

	/**
	 * Sets popover target action.
	 *
	 * @param value Popover target action.
	 */
	public set popoverTargetAction(value: string) {
		this.setAttribute('popovertargetaction', value);
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
	 * Selects the text.
	 */
	public select(): void {
		if (!this.#isSelectionSupported()) {
			return null;
		}

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
		if (!this.#isSelectionSupported()) {
			throw new this[PropertySymbol.window].DOMException(
				`The input element's type (${this.type}) does not support selection.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		this.#selectionEnd = Math.min(end, this.value.length);
		this.#selectionStart = Math.min(start, this.#selectionEnd);
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
		if (!this.#isSelectionSupported()) {
			throw new this[PropertySymbol.window].DOMException(
				`The input element's type (${this.type}) does not support selection.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

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
	 * Checks validity.
	 *
	 * @returns "true" if the field is valid.
	 */
	public checkValidity(): boolean {
		const valid =
			this.disabled ||
			this.readOnly ||
			this.type === 'hidden' ||
			this.type === 'reset' ||
			this.type === 'button' ||
			this[PropertySymbol.validity].valid;
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
	 * Steps up.
	 *
	 * @param [increment] Increment.
	 */
	public stepUp(increment?: number): void {
		const newValue = HTMLInputElementValueStepping.step(this.type, this.value, 1, increment);
		if (newValue !== null) {
			this.value = newValue;
		}
	}

	/**
	 * Steps down.
	 *
	 * @param [increment] Increment.
	 */
	public stepDown(increment?: number): void {
		const newValue = HTMLInputElementValueStepping.step(this.type, this.value, -1, increment);
		if (newValue !== null) {
			this.value = newValue;
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLInputElement {
		const clone = <HTMLInputElement>super[PropertySymbol.cloneNode](deep);
		clone[PropertySymbol.value] = this[PropertySymbol.value];
		clone[PropertySymbol.height] = this[PropertySymbol.height];
		clone[PropertySymbol.width] = this[PropertySymbol.width];
		clone[PropertySymbol.files] = <FileList>this[PropertySymbol.files].slice();
		clone.#selectionStart = this.#selectionStart;
		clone.#selectionEnd = this.#selectionEnd;
		clone.#selectionDirection = this.#selectionDirection;
		return clone;
	}

	/**
	 * @override
	 */
	public override dispatchEvent(event: Event): boolean {
		if (
			event[PropertySymbol.type] !== 'click' ||
			event[PropertySymbol.eventPhase] !== EventPhaseEnum.none ||
			!(event instanceof MouseEvent)
		) {
			return super.dispatchEvent(event);
		}

		// Do nothing if the input element is disabled and the event is a click event.
		if (this.disabled) {
			return false;
		}

		let previousCheckedValue: boolean | null = null;
		let previousIndeterminateValue: boolean = this[PropertySymbol.indeterminate];

		// The checkbox or radio button has to be checked before the click event is dispatched, so that event listeners can check the checked value.
		// However, the value has to be restored if preventDefault() is called on the click event.
		const type = this.type;

		if (type === 'checkbox' || type === 'radio') {
			previousCheckedValue = this.checked;
			this.#setChecked(type === 'checkbox' ? !previousCheckedValue : true);
			if (type === 'checkbox') {
				previousIndeterminateValue = this[PropertySymbol.indeterminate];
				this[PropertySymbol.indeterminate] = false;
			}
		}

		// Dispatches the event
		const returnValue = super.dispatchEvent(event);

		if (
			event[PropertySymbol.type] !== 'click' ||
			event[PropertySymbol.eventPhase] !== EventPhaseEnum.none ||
			!(event instanceof MouseEvent)
		) {
			return returnValue;
		}

		if (event[PropertySymbol.defaultPrevented]) {
			// Restore checked state if preventDefault() is triggered inside a listener of the click event.
			if (previousCheckedValue !== null) {
				this.#setChecked(previousCheckedValue);
				this[PropertySymbol.indeterminate] = previousIndeterminateValue;
			}
		} else {
			const type = this.type;

			if (type === 'checkbox' && this[PropertySymbol.isConnected]) {
				this.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
				this.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));
			} else if (type === 'radio' && !previousCheckedValue && this[PropertySymbol.isConnected]) {
				this.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
				this.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));
			} else if (type === 'submit' || type === 'reset') {
				const form = this.form;
				if (form) {
					if (type === 'submit' && this[PropertySymbol.isConnected]) {
						form.requestSubmit(this);
					} else if (type === 'reset') {
						form.reset();
					}
				}
			}
		}

		return returnValue;
	}

	/**
	 * Checks is selection is supported.
	 *
	 * @returns "true" if selection is supported.
	 */
	#isSelectionSupported(): boolean {
		const inputType = this.type;
		return (
			inputType === 'text' ||
			inputType === 'search' ||
			inputType === 'url' ||
			inputType === 'tel' ||
			inputType === 'password'
		);
	}

	/**
	 * Sets checked value.
	 *
	 * @param checked Checked.
	 */
	#setChecked(checked: boolean): void {
		this[PropertySymbol.checked] = checked;
		this[PropertySymbol.clearCache]();

		if (checked && this.type === 'radio' && this.name) {
			const root = <HTMLElement>(
				(<HTMLFormElement>this[PropertySymbol.formNode] || this.getRootNode())
			);
			const radioButtons = root.querySelectorAll(`input[type="radio"][name="${this.name}"]`);

			for (const radioButton of radioButtons) {
				if (radioButton !== this) {
					radioButton[PropertySymbol.checked] = false;
				}
			}
		}
	}
}
