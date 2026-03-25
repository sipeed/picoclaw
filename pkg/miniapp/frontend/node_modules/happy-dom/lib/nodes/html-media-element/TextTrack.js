import EventTarget from '../../event/EventTarget.js';
import TextTrackCueList from './TextTrackCueList.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import TextTrackKindEnum from './TextTrackKindEnum.js';
/**
 * TextTrack.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrack
 */
export default class TextTrack extends EventTarget {
    // Internal properties
    [PropertySymbol.kind] = TextTrackKindEnum.subtitles;
    [PropertySymbol.label] = '';
    [PropertySymbol.language] = '';
    [PropertySymbol.id] = '';
    [PropertySymbol.mode] = 'disabled';
    [PropertySymbol.cues] = new TextTrackCueList(PropertySymbol.illegalConstructor);
    [PropertySymbol.activeCues] = new TextTrackCueList(PropertySymbol.illegalConstructor);
    // Events
    oncuechange = null;
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
     * Returns the kind of the text track.
     *
     * @returns Kind.
     */
    get kind() {
        return this[PropertySymbol.kind];
    }
    /**
     * Returns the label of the text track.
     *
     * @returns Label.
     */
    get label() {
        return this[PropertySymbol.label];
    }
    /**
     * Returns the language of the text track.
     *
     * @returns Language.
     */
    get language() {
        return this[PropertySymbol.language];
    }
    /**
     * Returns the id of the text track.
     *
     * @returns Id.
     */
    get id() {
        return this[PropertySymbol.id];
    }
    /**
     * Returns the mode of the text track.
     *
     * @returns Mode.
     */
    get mode() {
        return this[PropertySymbol.mode];
    }
    /**
     * Sets the mode of the text track.
     *
     * @param mode Mode.
     */
    set mode(mode) {
        if (mode !== 'disabled' && mode !== 'showing') {
            // TODO: Browser outputs a warning here.
            return;
        }
        this[PropertySymbol.mode] = mode;
    }
    /**
     * Returns the list of cues in the track list.
     *
     * @returns List of cues.
     */
    get cues() {
        if (this[PropertySymbol.mode] === 'disabled') {
            return null;
        }
        return this[PropertySymbol.cues];
    }
    /**
     * Returns the list of active cues in the track list.
     *
     * @returns List of active cues.
     */
    get activeCues() {
        if (this[PropertySymbol.mode] === 'disabled') {
            return null;
        }
        return this[PropertySymbol.activeCues];
    }
    /**
     * Adds a cue to the track list.
     *
     * @param cue Text track cue.
     */
    addCue(cue) {
        if (this[PropertySymbol.cues].includes(cue)) {
            return;
        }
        cue[PropertySymbol.track] = this;
        this[PropertySymbol.cues].push(cue);
    }
    /**
     * Removes a cue from the track list.
     *
     * @param cue Text track cue.
     */
    removeCue(cue) {
        const index = this[PropertySymbol.cues].indexOf(cue);
        if (index !== -1) {
            cue[PropertySymbol.track] = null;
            this[PropertySymbol.cues].splice(index, 1);
        }
    }
}
//# sourceMappingURL=TextTrack.js.map