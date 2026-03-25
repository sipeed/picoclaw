import Event from '../Event.js';
/**
 *
 */
export default class ProgressEvent extends Event {
    lengthComputable;
    loaded;
    total;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type);
        this.lengthComputable = eventInit?.lengthComputable ?? false;
        this.loaded = eventInit?.loaded ?? 0;
        this.total = eventInit?.total ?? 0;
    }
}
//# sourceMappingURL=ProgressEvent.js.map