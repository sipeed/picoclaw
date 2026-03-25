import UIEvent from '../UIEvent.js';
/**
 *
 */
export default class FocusEvent extends UIEvent {
    relatedTarget;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.relatedTarget = eventInit?.relatedTarget ?? null;
    }
}
//# sourceMappingURL=FocusEvent.js.map