import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import Event from '../../event/Event.js';
import HTMLInputElement from '../html-input-element/HTMLInputElement.js';
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
    cloneNode: (deep?: boolean) => HTMLLabelElement;
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
     * Returns an HTML element representing the control with which the label is associated.
     *
     * @returns Control element.
     */
    get control(): HTMLInputElement | HTMLButtonElement | HTMLMeterElement | HTMLOutputElement | HTMLProgressElement | HTMLSelectElement | HTMLTextAreaElement | null;
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form(): HTMLFormElement | null;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLLabelElement;
    /**
     * @override
     */
    dispatchEvent(event: Event): boolean;
}
//# sourceMappingURL=HTMLLabelElement.d.ts.map