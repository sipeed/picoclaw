import Event from '../../event/Event.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML Dialog Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDialogElement
 */
export default class HTMLDialogElement extends HTMLElement {
    // Internal properties
    [PropertySymbol.returnValue] = '';
    // Events
    oncancel = null;
    onclose = null;
    /**
     * Returns return value.
     *
     * @returns Return value.
     */
    get returnValue() {
        return this[PropertySymbol.returnValue];
    }
    /**
     * Sets return value.
     *
     * @param value Return value.
     */
    set returnValue(value) {
        this[PropertySymbol.returnValue] = value;
    }
    /**
     * Sets the "open" attribute.
     *
     * @param open Open.
     */
    set open(open) {
        if (open) {
            this.setAttribute('open', '');
        }
        else {
            this.removeAttribute('open');
        }
    }
    /**
     * Returns open.
     *
     * @returns Open.
     */
    get open() {
        return this.getAttribute('open') !== null;
    }
    /**
     * Closes the dialog.
     *
     * @param [returnValue] ReturnValue.
     */
    close(returnValue = '') {
        const wasOpen = this.open;
        this.removeAttribute('open');
        this.returnValue = returnValue;
        if (wasOpen) {
            this.dispatchEvent(new Event('close'));
        }
    }
    /**
     * Shows the modal.
     */
    showModal() {
        this.setAttribute('open', '');
    }
    /**
     * Shows the dialog.
     */
    show() {
        this.setAttribute('open', '');
    }
}
//# sourceMappingURL=HTMLDialogElement.js.map