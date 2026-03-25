import MouseEvent from './MouseEvent.js';
import IPointerEventInit from './IPointerEventInit.js';

/**
 *
 */
export default class PointerEvent extends MouseEvent {
	public readonly pointerId: number;
	public readonly width: number;
	public readonly height: number;
	public readonly pressure: number;
	public readonly tangentialPressure: number;
	public readonly tiltX: number;
	public readonly tiltY: number;
	public readonly twist: number;
	public readonly altitudeAngle: number;
	public readonly azimuthAngle: number;
	public readonly pointerType: string;
	public readonly isPrimary: boolean;
	public readonly coalescedEvents: PointerEvent[];
	public readonly predictedEvents: PointerEvent[];

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IPointerEventInit | null = null) {
		super(type, eventInit);

		this.pointerId = eventInit?.pointerId ?? 0;
		this.width = eventInit?.width ?? 1;
		this.height = eventInit?.height ?? 1;
		this.pressure = eventInit?.pressure ?? 0;
		this.tangentialPressure = eventInit?.tangentialPressure ?? 0;
		this.tiltX = eventInit?.tiltX ?? 0;
		this.tiltY = eventInit?.tiltY ?? 0;
		this.twist = eventInit?.twist ?? 0;
		this.altitudeAngle = eventInit?.altitudeAngle ?? 0;
		this.azimuthAngle = eventInit?.azimuthAngle ?? 0;
		this.pointerType = eventInit?.pointerType ?? '';
		this.isPrimary = eventInit?.isPrimary ?? false;
		this.coalescedEvents = eventInit?.coalescedEvents ?? [];
		this.predictedEvents = eventInit?.predictedEvents ?? [];
	}

	public getCoalescedEvents = (): PointerEvent[] => this.coalescedEvents;

	public getPredictedEvents = (): PointerEvent[] => this.predictedEvents;
}
