import ITouchEventInit from './ITouchEventInit.cjs';
import UIEvent from '../UIEvent.cjs';
import Touch from '../Touch.cjs';
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