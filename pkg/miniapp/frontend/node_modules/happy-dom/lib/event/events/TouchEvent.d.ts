import ITouchEventInit from './ITouchEventInit.js';
import UIEvent from '../UIEvent.js';
import Touch from '../Touch.js';
/**
 *
 */
export default class TouchEvent extends UIEvent {
    readonly altKey: boolean;
    readonly changedTouches: Touch[];
    readonly ctrlKey: boolean;
    readonly metaKey: boolean;
    readonly shiftKey: boolean;
    readonly targetTouches: Touch[];
    readonly touches: Touch[];
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: ITouchEventInit | null);
}
//# sourceMappingURL=TouchEvent.d.ts.map