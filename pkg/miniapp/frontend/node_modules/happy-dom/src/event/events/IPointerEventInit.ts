import IMouseEventInit from './IMouseEventInit.js';
import PointerEvent from './PointerEvent.js';

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
