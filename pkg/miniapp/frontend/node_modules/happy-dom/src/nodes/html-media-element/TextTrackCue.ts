import EventTarget from '../../event/EventTarget.js';
import Event from '../../event/Event.js';
import TextTrack from './TextTrack.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * TextTrackCue.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrackCue
 */
export default abstract class TextTrackCue extends EventTarget {
	// Public properties
	public id: string = '';
	public startTime: number = 0;
	public endTime: number = 0;
	public pauseOnExit: boolean = false;

	// Internal properties
	public [PropertySymbol.track]: TextTrack | null = null;

	// Events
	public onenter: (event: Event) => void = null;
	public onexit: (event: Event) => void = null;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 */
	constructor(illegalConstructorSymbol: symbol) {
		super();
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		if (!this[PropertySymbol.window]) {
			throw new TypeError(
				`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`
			);
		}
	}

	/**
	 * Returns the owner track.
	 *
	 * @returns TextTrack.
	 */
	public get track(): TextTrack {
		return this[PropertySymbol.track];
	}
}
