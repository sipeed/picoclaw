import MouseEvent from './MouseEvent.cjs';
import IPointerEventInit from './IPointerEventInit.cjs';
/**
 *
 */
export default class PointerEvent extends MouseEvent {
    readonly pointerId: number;
    readonly width: number;
    readonly height: number;
    readonly pressure: number;
    readonly tangentialPressure: number;
    readonly tiltX: number;
    readonly tiltY: number;
    readonly twist: number;
    readonly altitudeAngle: number;
    readonly azimuthAngle: number;
    readonly pointerType: string;
    readonly isPrimary: boolean;
    readonly coalescedEvents: PointerEvent[];
    readonly predictedEvents: PointerEvent[];
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IPointerEventInit | null);
    getCoalescedEvents: () => PointerEvent[];
    getPredictedEvents: () => PointerEvent[];
}
//# sourceMappingURL=PointerEvent.d.ts.map