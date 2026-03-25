import ITouchEventInit from './ITouchEventInit.js';
import UIEvent from '../UIEvent.js';
import Touch from '../Touch.js';

/**
 *
 */
export default class TouchEvent extends UIEvent {
	public readonly altKey: boolean;
	public readonly changedTouches: Touch[];
	public readonly ctrlKey: boolean;
	public readonly metaKey: boolean;
	public readonly shiftKey: boolean;
	public readonly targetTouches: Touch[];
	public readonly touches: Touch[];

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: ITouchEventInit | null = null) {
		super(type, eventInit);

		this.altKey = eventInit?.altKey ?? false;
		this.changedTouches = eventInit?.changedTouches ?? [];
		this.ctrlKey = eventInit?.ctrlKey ?? false;
		this.metaKey = eventInit?.metaKey ?? false;
		this.shiftKey = eventInit?.shiftKey ?? false;
		this.targetTouches = eventInit?.targetTouches ?? [];
		this.touches = eventInit?.touches ?? [];
	}
}
