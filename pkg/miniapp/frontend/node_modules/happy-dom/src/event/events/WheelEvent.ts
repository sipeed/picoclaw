import UIEvent from '../UIEvent.js';
import IWheelEventInit from './IWheelEventInit.js';

/**
 *
 */
export default class WheelEvent extends UIEvent {
	public static DOM_DELTA_PIXEL = 0;
	public static DOM_DELTA_LINE = 1;
	public static DOM_DELTA_PAGE = 2;
	public readonly deltaX: number;
	public readonly deltaY: number;
	public readonly deltaZ: number;
	public readonly deltaMode: number;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IWheelEventInit | null = null) {
		super(type, eventInit);

		this.deltaX = eventInit?.deltaX ?? 0;
		this.deltaY = eventInit?.deltaY ?? 0;
		this.deltaZ = eventInit?.deltaZ ?? 0;
		this.deltaMode = eventInit?.deltaMode ?? 0;
	}
}
