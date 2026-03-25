import Event from '../Event.js';
import IMediaQueryListInit from './IMediaQueryListInit.js';

/**
 *
 */
export default class MediaQueryListEvent extends Event {
	public readonly matches: boolean;
	public readonly media: string;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IMediaQueryListInit | null = null) {
		super(type, eventInit);

		this.matches = eventInit?.matches ?? false;
		this.media = eventInit?.media ?? '';
	}
}
