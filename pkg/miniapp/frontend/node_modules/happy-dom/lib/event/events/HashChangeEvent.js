import Event from '../Event.js';
/**
 * Hash change event.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HashChangeEvent
 */
export default class HashChangeEvent extends Event {
    newURL;
    oldURL;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.newURL = eventInit?.newURL ?? '';
        this.oldURL = eventInit?.oldURL ?? '';
    }
}
//# sourceMappingURL=HashChangeEvent.js.map