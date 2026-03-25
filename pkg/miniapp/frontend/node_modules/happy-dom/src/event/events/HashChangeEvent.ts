import Event from '../Event.js';
import IHashChangeEventInit from './IHashChangeEventInit.js';

/**
 * Hash change event.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HashChangeEvent
 */
export default class HashChangeEvent extends Event {
	public readonly newURL: string;
	public readonly oldURL: string;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IHashChangeEventInit | null = null) {
		super(type, eventInit);

		this.newURL = eventInit?.newURL ?? '';
		this.oldURL = eventInit?.oldURL ?? '';
	}
}
