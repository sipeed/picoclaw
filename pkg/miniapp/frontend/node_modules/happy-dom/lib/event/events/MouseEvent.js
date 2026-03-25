import UIEvent from '../UIEvent.js';
/**
 *
 */
export default class MouseEvent extends UIEvent {
    altKey;
    button;
    buttons;
    clientX;
    clientY;
    ctrlKey;
    metaKey;
    movementX;
    movementY;
    offsetX;
    offsetY;
    region;
    relatedTarget;
    screenX;
    screenY;
    shiftKey;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.altKey = eventInit?.altKey ?? false;
        this.button = eventInit?.button ?? 0;
        this.buttons = eventInit?.buttons ?? 0;
        this.clientX = eventInit?.clientX ?? 0;
        this.clientY = eventInit?.clientY ?? 0;
        this.ctrlKey = eventInit?.ctrlKey ?? false;
        this.metaKey = eventInit?.metaKey ?? false;
        this.movementX = eventInit?.movementX ?? 0;
        this.movementY = eventInit?.movementY ?? 0;
        this.region = eventInit?.region ?? '';
        this.relatedTarget = eventInit?.relatedTarget ?? null;
        this.screenX = eventInit?.screenX ?? 0;
        this.screenY = eventInit?.screenY ?? 0;
        this.shiftKey = eventInit?.shiftKey ?? false;
    }
}
//# sourceMappingURL=MouseEvent.js.map