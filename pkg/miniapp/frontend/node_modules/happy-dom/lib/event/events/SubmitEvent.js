import Event from '../Event.js';
/**
 * An event triggered by form submit buttons.
 */
export default class SubmitEvent extends Event {
    submitter;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.submitter = eventInit?.submitter ?? null;
    }
}
//# sourceMappingURL=SubmitEvent.js.map