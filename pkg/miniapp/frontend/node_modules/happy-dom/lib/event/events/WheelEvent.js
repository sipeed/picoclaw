import UIEvent from '../UIEvent.js';
/**
 *
 */
export default class WheelEvent extends UIEvent {
    static DOM_DELTA_PIXEL = 0;
    static DOM_DELTA_LINE = 1;
    static DOM_DELTA_PAGE = 2;
    deltaX;
    deltaY;
    deltaZ;
    deltaMode;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.deltaX = eventInit?.deltaX ?? 0;
        this.deltaY = eventInit?.deltaY ?? 0;
        this.deltaZ = eventInit?.deltaZ ?? 0;
        this.deltaMode = eventInit?.deltaMode ?? 0;
    }
}
//# sourceMappingURL=WheelEvent.js.map