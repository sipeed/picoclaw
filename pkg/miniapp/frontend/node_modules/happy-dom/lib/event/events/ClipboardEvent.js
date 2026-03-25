import Event from '../Event.js';
/**
 *
 */
export default class ClipboardEvent extends Event {
    clipboardData;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.clipboardData = eventInit?.clipboardData ?? null;
    }
}
//# sourceMappingURL=ClipboardEvent.js.map