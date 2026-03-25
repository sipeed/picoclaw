import Event from './Event.js';
/**
 *
 */
export default class UIEvent extends Event {
    static NONE = 0;
    static CAPTURING_PHASE = 1;
    static AT_TARGET = 2;
    static BUBBLING_PHASE = 3;
    detail;
    layerX = 0;
    layerY = 0;
    pageX = 0;
    pageY = 0;
    view;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.detail = eventInit?.detail ?? 0;
        this.view = eventInit?.view ?? null;
    }
}
//# sourceMappingURL=UIEvent.js.map