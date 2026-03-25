import Event from '../Event.js';
/**
 *
 */
export default class MediaQueryListEvent extends Event {
    matches;
    media;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.matches = eventInit?.matches ?? false;
        this.media = eventInit?.media ?? '';
    }
}
//# sourceMappingURL=MediaQueryListEvent.js.map