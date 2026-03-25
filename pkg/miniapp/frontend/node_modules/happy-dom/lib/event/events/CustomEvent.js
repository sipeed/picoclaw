import Event from '../Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 *
 */
export default class CustomEvent extends Event {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    [PropertySymbol.detail];
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this[PropertySymbol.detail] = eventInit?.detail ?? null;
    }
    /**
     * Returns detail.
     *
     * @returns Detail.
     */
    get detail() {
        return this[PropertySymbol.detail];
    }
    /**
     * Init event.
     *
     * @deprecated
     * @param type Type.
     * @param [bubbles=false] "true" if it bubbles.
     * @param [cancelable=false] "true" if it cancelable.
     * @param [detail=null] Custom event detail.
     */
    initCustomEvent(type, bubbles = false, cancelable = false, detail = null) {
        this[PropertySymbol.type] = type;
        this[PropertySymbol.bubbles] = bubbles;
        this[PropertySymbol.cancelable] = cancelable;
        this[PropertySymbol.detail] = detail;
    }
}
//# sourceMappingURL=CustomEvent.js.map