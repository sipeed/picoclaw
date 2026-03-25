import IMouseEventInit from './IMouseEventInit.cjs';
import PointerEvent from './PointerEvent.cjs';
export default interface IPointerEventInit extends IMouseEventInit {
    pointerId?: number;
    width?: number;
    height?: number;
    pressure?: number;
    tangentialPressure?: number;
    tiltX?: number;
    tiltY?: number;
    twist?: number;
    altitudeAngle?: number;
    azimuthAngle?: number;
    pointerType?: string;
    isPrimary?: boolean;
    coalescedEvents?: PointerEvent[];
    predictedEvents?: PointerEvent[];
}
//# sourceMappingURL=IPointerEventInit.d.ts.map