import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import ValidityState from '../../validity-state/ValidityState.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import { URL } from 'url';
import MouseEvent from '../../event/events/MouseEvent.js';
import NodeList from '../node/NodeList.js';

const BUTTON_TYPES = ['submit', 'reset', 'button', 'menu'];

/**
 * HTML Button Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLButtonElement.
 */
export default class HTMLButtonElement extends HTMLElement {
	public [PropertySymbol.validationMessage] = '';
	public [PropertySymbol.validity] = new ValidityState(this);
	public [PropertySymbol.formNode]: HTMLFormElement | null = null;
	public [PropertySymbol.popoverTargetElement]: HTMLElement | null = null;

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
		return this.getAttribute('value');
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		this.setAttribute('value', value);
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
	 * Returns type
	 *
	 * @returns Type
	 */
	public get type(): string {
		const type = this.getAttribute('type');
		if (type === null || !BUTTON_TYPES.includes(type)) {
			return 'submit';
		}
		return type;
	}

	/**
	 * Sets type
	 *
	 * @param value Type
	 */
	public set type(value: string) {
		this.setAttribute('type', value);
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
	 * Returns the associated label elements.
	 *
	 * @returns Label elements.
	 */
	public get labels(): NodeList<HTMLLabelElement> {
		return HTMLLabelElementUtility.getAssociatedLabelElements(this);
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
	 * Checks validity.
	 *
	 * @returns "true" if the field is valid.
	 */
	public checkValidity(): boolean {
		const valid =
			this.disabled ||
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

	/**
	 * @override
	 */
	public override dispatchEvent(event: Event): boolean {
		if (
			event.type === 'click' &&
			event instanceof MouseEvent &&
			event.eventPhase === EventPhaseEnum.none &&
			this.disabled
		) {
			return false;
		}

		const returnValue = super.dispatchEvent(event);

		if (
			!event[PropertySymbol.defaultPrevented] &&
			event.type === 'click' &&
			event.eventPhase === EventPhaseEnum.none &&
			event instanceof MouseEvent
		) {
			const type = this.type;
			if (type === 'submit' || type === 'reset') {
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
}
