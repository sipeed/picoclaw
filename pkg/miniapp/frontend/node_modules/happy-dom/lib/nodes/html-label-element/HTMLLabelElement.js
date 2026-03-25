import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import MouseEvent from '../../event/events/MouseEvent.js';
/**
 * HTML Label Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLabelElement.
 */
export default class HTMLLabelElement extends HTMLElement {
    /**
     * Returns a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @returns ID of the labeled control.
     */
    get htmlFor() {
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
    set htmlFor(htmlFor) {
        this.setAttribute('for', htmlFor);
    }
    /**
     * Returns an HTML element representing the control with which the label is associated.
     *
     * @returns Control element.
     */
    get control() {
        const htmlFor = this.getAttribute('for');
        if (htmlFor !== null) {
            if (!htmlFor || !this[PropertySymbol.isConnected]) {
                return null;
            }
            const control = (this[PropertySymbol.rootNode].getElementById(htmlFor));
            if (control) {
                switch (control[PropertySymbol.tagName]) {
                    case 'INPUT':
                        return control.type !== 'hidden' ? control : null;
                    case 'BUTTON':
                    case 'METER':
                    case 'OUTPUT':
                    case 'PROGRESS':
                    case 'SELECT':
                    case 'TEXTAREA':
                        return control;
                    default:
                        return null;
                }
            }
        }
        return (this.querySelector('button,input:not([type="hidden"]),meter,output,progress,select,textarea'));
    }
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        return this.control?.form || null;
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
    /**
     * @override
     */
    dispatchEvent(event) {
        const returnValue = super.dispatchEvent(event);
        if (!event[PropertySymbol.defaultPrevented] &&
            event.type === 'click' &&
            event.eventPhase === EventPhaseEnum.none &&
            event instanceof MouseEvent) {
            const control = this.control;
            if (control && event.target !== control) {
                control.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
            }
        }
        return returnValue;
    }
}
//# sourceMappingURL=HTMLLabelElement.js.map