import UIEvent from '../UIEvent.js';
/**
 *
 */
export default class ErrorEvent extends UIEvent {
    message;
    filename;
    lineno;
    colno;
    error;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.message = eventInit?.message ?? '';
        this.filename = eventInit?.filename ?? '';
        this.lineno = eventInit?.lineno ?? 0;
        this.colno = eventInit?.colno ?? 0;
        this.error = eventInit?.error ?? null;
    }
}
//# sourceMappingURL=ErrorEvent.js.map