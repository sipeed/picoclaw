import UIEvent from '../UIEvent.js';
import EventTarget from '../EventTarget.js';
import IMouseEventInit from './IMouseEventInit.js';

/**
 *
 */
export default class MouseEvent extends UIEvent {
	public readonly altKey: boolean;
	public readonly button: number;
	public readonly buttons: number;
	public readonly clientX: number;
	public readonly clientY: number;
	public readonly ctrlKey: boolean;
	public readonly metaKey: boolean;
	public readonly movementX: number;
	public readonly movementY: number;
	public readonly offsetX: number;
	public readonly offsetY: number;
	public readonly region: string;
	public readonly relatedTarget: EventTarget | null;
	public readonly screenX: number;
	public readonly screenY: number;
	public readonly shiftKey: boolean;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IMouseEventInit | null = null) {
		super(type, eventInit);

		this.altKey = eventInit?.altKey ?? false;
		this.button = eventInit?.button ?? 0;
		this.buttons = eventInit?.buttons ?? 0;
		this.clientX = eventInit?.clientX ?? 0;
		this.clientY = eventInit?.clientY ?? 0;
		this.ctrlKey = eventInit?.ctrlKey ?? false;
		this.metaKey = eventInit?.metaKey ?? false;
		this.movementX = eventInit?.movementX ?? 0;
		this.movementY = eventInit?.movementY ?? 0;
		this.region = eventInit?.region ?? '';
		this.relatedTarget = eventInit?.relatedTarget ?? null;
		this.screenX = eventInit?.screenX ?? 0;
		this.screenY = eventInit?.screenY ?? 0;
		this.shiftKey = eventInit?.shiftKey ?? false;
	}
}
