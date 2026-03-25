import TextTrackCue from './TextTrackCue.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 *
 */
export default class TextTrackCueList extends Array<TextTrackCue> {
	/**
	 * Constructor.
	 *
	 * @param [illegalConstructorSymbol] Illegal constructor symbol.
	 */
	constructor(illegalConstructorSymbol?: symbol) {
		super();
		// "illegalConstructorSymbol" can be "1" when calling the "splice()" method
		if (
			<number>(<unknown>illegalConstructorSymbol) !== 1 &&
			illegalConstructorSymbol !== PropertySymbol.illegalConstructor
		) {
			throw new TypeError('Illegal constructor');
		}
	}

	/**
	 * Returns the first TextTrackCue object with the identifier passed to it.
	 *
	 * @param id Text track cue identifier.
	 */
	public getCueById(id: string): TextTrackCue | null {
		for (const cue of this) {
			if (cue.id === id) {
				return cue;
			}
		}
		return null;
	}
}
