import Event from '../../event/Event.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import MouseEvent from '../../event/events/MouseEvent.js';
/**
 * HTMLDetailsElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDetailsElement
 */
export default class HTMLDetailsElement extends HTMLElement {
    // Events
    ontoggle = null;
    /**
     * Returns the open attribute.
     */
    get open() {
        return this.getAttribute('open') !== null;
    }
    /**
     * Sets the open attribute.
     *
     * @param open New value.
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
     * @override
     */
    [PropertySymbol.onSetAttribute](attribute, replacedAttribute) {
        super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
        if (attribute[PropertySymbol.name] === 'open') {
            if (attribute[PropertySymbol.value] !== replacedAttribute?.[PropertySymbol.value]) {
                this.dispatchEvent(new Event('toggle'));
            }
        }
    }
    /**
     * @override
     */
    [PropertySymbol.onRemoveAttribute](removedAttribute) {
        super[PropertySymbol.onRemoveAttribute](removedAttribute);
        if (removedAttribute && removedAttribute[PropertySymbol.name] === 'open') {
            this.dispatchEvent(new Event('toggle'));
        }
    }
    /**
     * @override
     */
    dispatchEvent(event) {
        const returnValue = super.dispatchEvent(event);
        if (!event[PropertySymbol.defaultPrevented] &&
            event[PropertySymbol.target]?.[PropertySymbol.localName] === 'summary' &&
            event.type === 'click' &&
            event.eventPhase === EventPhaseEnum.bubbling &&
            event instanceof MouseEvent) {
            this.open = !this.open;
        }
        return returnValue;
    }
}
//# sourceMappingURL=HTMLDetailsElement.js.map