import UIEvent from '../UIEvent.js';
import EventTarget from '../EventTarget.js';
import IFocusEventInit from './IFocusEventInit.js';

/**
 *
 */
export default class FocusEvent extends UIEvent {
	public readonly relatedTarget: EventTarget | null;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IFocusEventInit | null = null) {
		super(type, eventInit);

		this.relatedTarget = eventInit?.relatedTarget ?? null;
	}
}
