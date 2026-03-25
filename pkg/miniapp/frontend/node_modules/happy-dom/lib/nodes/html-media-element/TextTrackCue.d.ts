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
    id: string;
    startTime: number;
    endTime: number;
    pauseOnExit: boolean;
    [PropertySymbol.track]: TextTrack | null;
    onenter: (event: Event) => void;
    onexit: (event: Event) => void;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol: symbol);
    /**
     * Returns the owner track.
     *
     * @returns TextTrack.
     */
    get track(): TextTrack;
}
//# sourceMappingURL=TextTrackCue.d.ts.map