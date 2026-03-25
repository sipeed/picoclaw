import Event from '../Event.js';
/**
 * Media Stream Track Event.
 */
export default class MediaStreamTrackEvent extends Event {
    track;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.track = eventInit?.track ?? null;
    }
}
//# sourceMappingURL=MediaStreamTrackEvent.js.map