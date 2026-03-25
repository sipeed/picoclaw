import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import Event from '../../event/Event.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import HTMLInputElement from '../html-input-element/HTMLInputElement.js';
import Document from '../document/Document.js';
import MouseEvent from '../../event/events/MouseEvent.js';
import HTMLButtonElement from '../html-button-element/HTMLButtonElement.js';
import HTMLMeterElement from '../html-meter-element/HTMLMeterElement.js';
import HTMLOutputElement from '../html-output-element/HTMLOutputElement.js';
import HTMLProgressElement from '../html-progress-element/HTMLProgressElement.js';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.js';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.js';

/**
 * HTML Label Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLabelElement.
 */
export default class HTMLLabelElement extends HTMLElement {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLLabelElement;

	/**
	 * Returns a string containing the ID of the labeled control. This reflects the "for" attribute.
	 *
	 * @returns ID of the labeled control.
	 */
	public get htmlFor(): string {
		const htmlFor = this.getAttribute('for');
		if (htmlFor !== null) {
			return htmlFor;
		}
		return htmlFor !== null ? htmlFor : '';
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
	 * Returns an HTML element representing the control with which the label is associated.
	 *
	 * @returns Control element.
	 */
	public get control():
		| HTMLInputElement
		| HTMLButtonElement
		| HTMLMeterElement
		| HTMLOutputElement
		| HTMLProgressElement
		| HTMLSelectElement
		| HTMLTextAreaElement
		| null {
		const htmlFor = this.getAttribute('for');
		if (htmlFor !== null) {
			if (!htmlFor || !this[PropertySymbol.isConnected]) {
				return null;
			}
			const control = <HTMLElement | null>(
				(<Document>this[PropertySymbol.rootNode]).getElementById(htmlFor)
			);
			if (control) {
				switch (control[PropertySymbol.tagName]) {
					case 'INPUT':
						return (<HTMLInputElement>control).type !== 'hidden' ? <HTMLInputElement>control : null;
					case 'BUTTON':
					case 'METER':
					case 'OUTPUT':
					case 'PROGRESS':
					case 'SELECT':
					case 'TEXTAREA':
						return <HTMLInputElement>control;
					default:
						return null;
				}
			}
		}
		return <HTMLInputElement | null>(
			this.querySelector('button,input:not([type="hidden"]),meter,output,progress,select,textarea')
		);
	}

	/**
	 * Returns the parent form element.
	 *
	 * @returns Form.
	 */
	public get form(): HTMLFormElement | null {
		return (<HTMLInputElement>this.control)?.form || null;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLLabelElement {
		return <HTMLLabelElement>super[PropertySymbol.cloneNode](deep);
	}

	/**
	 * @override
	 */
	public override dispatchEvent(event: Event): boolean {
		const returnValue = super.dispatchEvent(event);

		if (
			!event[PropertySymbol.defaultPrevented] &&
			event.type === 'click' &&
			event.eventPhase === EventPhaseEnum.none &&
			event instanceof MouseEvent
		) {
			const control = this.control;
			if (control && event.target !== control) {
				control.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
			}
		}

		return returnValue;
	}
}
