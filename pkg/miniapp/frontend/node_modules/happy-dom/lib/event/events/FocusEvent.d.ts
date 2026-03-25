import UIEvent from '../UIEvent.js';
import EventTarget from '../EventTarget.js';
import IFocusEventInit from './IFocusEventInit.js';
/**
 *
 */
export default class FocusEvent extends UIEvent {
    readonly relatedTarget: EventTarget | null;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IFocusEventInit | null);
}
//# sourceMappingURL=FocusEvent.d.ts.map