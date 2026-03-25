import EventTarget from '../../event/EventTarget.js';
import Event from '../../event/Event.js';
import TextTrackCue from './TextTrackCue.js';
import TextTrackCueList from './TextTrackCueList.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import TextTrackKindEnum from './TextTrackKindEnum.js';
/**
 * TextTrack.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrack
 */
export default class TextTrack extends EventTarget {
    [PropertySymbol.kind]: TextTrackKindEnum;
    [PropertySymbol.label]: string;
    [PropertySymbol.language]: string;
    [PropertySymbol.id]: string;
    [PropertySymbol.mode]: 'disabled' | 'showing';
    [PropertySymbol.cues]: TextTrackCueList;
    [PropertySymbol.activeCues]: TextTrackCueList;
    oncuechange: (event: Event) => void;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol: symbol);
    /**
     * Returns the kind of the text track.
     *
     * @returns Kind.
     */
    get kind(): TextTrackKindEnum;
    /**
     * Returns the label of the text track.
     *
     * @returns Label.
     */
    get label(): string;
    /**
     * Returns the language of the text track.
     *
     * @returns Language.
     */
    get language(): string;
    /**
     * Returns the id of the text track.
     *
     * @returns Id.
     */
    get id(): string;
    /**
     * Returns the mode of the text track.
     *
     * @returns Mode.
     */
    get mode(): 'disabled' | 'showing';
    /**
     * Sets the mode of the text track.
     *
     * @param mode Mode.
     */
    set mode(mode: 'disabled' | 'showing');
    /**
     * Returns the list of cues in the track list.
     *
     * @returns List of cues.
     */
    get cues(): TextTrackCueList | null;
    /**
     * Returns the list of active cues in the track list.
     *
     * @returns List of active cues.
     */
    get activeCues(): TextTrackCueList | null;
    /**
     * Adds a cue to the track list.
     *
     * @param cue Text track cue.
     */
    addCue(cue: TextTrackCue): void;
    /**
     * Removes a cue from the track list.
     *
     * @param cue Text track cue.
     */
    removeCue(cue: TextTrackCue): void;
}
//# sourceMappingURL=TextTrack.d.ts.map