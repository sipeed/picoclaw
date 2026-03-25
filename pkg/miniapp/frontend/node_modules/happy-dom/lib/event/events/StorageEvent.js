import Event from '../Event.js';
/**
 *
 */
export default class StorageEvent extends Event {
    key;
    oldValue;
    newValue;
    url;
    storageArea;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type);
        this.key = eventInit?.key ?? null;
        this.oldValue = eventInit?.oldValue ?? null;
        this.newValue = eventInit?.newValue ?? null;
        this.url = eventInit?.url ?? '';
        this.storageArea = eventInit?.storageArea ?? null;
    }
}
//# sourceMappingURL=StorageEvent.js.map