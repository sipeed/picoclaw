import UIEvent from '../UIEvent.js';
/**
 *
 */
export default class KeyboardEvent extends UIEvent {
    static DOM_KEY_LOCATION_STANDARD = 0;
    static DOM_KEY_LOCATION_LEFT = 1;
    static DOM_KEY_LOCATION_RIGHT = 2;
    static DOM_KEY_LOCATION_NUMPAD = 3;
    altKey;
    code;
    ctrlKey;
    isComposing;
    key;
    location;
    metaKey;
    repeat;
    shiftKey;
    /**
     * @deprecated
     */
    keyCode;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.altKey = eventInit?.altKey ?? false;
        this.code = eventInit?.code ?? '';
        this.ctrlKey = eventInit?.ctrlKey ?? false;
        this.isComposing = eventInit?.isComposing ?? false;
        this.key = eventInit?.key ?? '';
        this.location = eventInit?.location ?? 0;
        this.metaKey = eventInit?.metaKey ?? false;
        this.repeat = eventInit?.repeat ?? false;
        this.shiftKey = eventInit?.shiftKey ?? false;
        this.keyCode = eventInit?.keyCode ?? 0;
    }
}
//# sourceMappingURL=KeyboardEvent.js.map