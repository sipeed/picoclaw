import Event from '../Event.js';
/**
 * Message event.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MessageEvent
 */
export default class MessageEvent extends Event {
    data;
    origin;
    lastEventId;
    source;
    ports;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.data = eventInit?.data ?? null;
        this.origin = eventInit?.origin ?? '';
        this.lastEventId = eventInit?.lastEventId ?? '';
        this.source = eventInit?.source ?? null;
        this.ports = eventInit?.ports ?? [];
    }
}
//# sourceMappingURL=MessageEvent.js.map