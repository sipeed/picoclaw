import UIEvent from '../UIEvent.cjs';
import EventTarget from '../EventTarget.cjs';
import IFocusEventInit from './IFocusEventInit.cjs';
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