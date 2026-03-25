import UIEvent from '../UIEvent.js';
/**
 *
 */
export default class TouchEvent extends UIEvent {
    altKey;
    changedTouches;
    ctrlKey;
    metaKey;
    shiftKey;
    targetTouches;
    touches;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.altKey = eventInit?.altKey ?? false;
        this.changedTouches = eventInit?.changedTouches ?? [];
        this.ctrlKey = eventInit?.ctrlKey ?? false;
        this.metaKey = eventInit?.metaKey ?? false;
        this.shiftKey = eventInit?.shiftKey ?? false;
        this.targetTouches = eventInit?.targetTouches ?? [];
        this.touches = eventInit?.touches ?? [];
    }
}
//# sourceMappingURL=TouchEvent.js.map