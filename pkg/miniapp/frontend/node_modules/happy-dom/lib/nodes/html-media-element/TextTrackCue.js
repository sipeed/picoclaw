import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * TextTrackCue.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrackCue
 */
export default class TextTrackCue extends EventTarget {
    // Public properties
    id = '';
    startTime = 0;
    endTime = 0;
    pauseOnExit = false;
    // Internal properties
    [PropertySymbol.track] = null;
    // Events
    onenter = null;
    onexit = null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol) {
        super();
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        if (!this[PropertySymbol.window]) {
            throw new TypeError(`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`);
        }
    }
    /**
     * Returns the owner track.
     *
     * @returns TextTrack.
     */
    get track() {
        return this[PropertySymbol.track];
    }
}
//# sourceMappingURL=TextTrackCue.js.map