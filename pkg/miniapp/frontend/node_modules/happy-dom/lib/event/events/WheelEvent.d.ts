import UIEvent from '../UIEvent.js';
import IWheelEventInit from './IWheelEventInit.js';
/**
 *
 */
export default class WheelEvent extends UIEvent {
    static DOM_DELTA_PIXEL: number;
    static DOM_DELTA_LINE: number;
    static DOM_DELTA_PAGE: number;
    readonly deltaX: number;
    readonly deltaY: number;
    readonly deltaZ: number;
    readonly deltaMode: number;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IWheelEventInit | null);
}
//# sourceMappingURL=WheelEvent.d.ts.map