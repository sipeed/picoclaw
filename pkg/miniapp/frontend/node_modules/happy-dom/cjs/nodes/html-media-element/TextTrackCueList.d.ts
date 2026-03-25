import TextTrackCue from './TextTrackCue.cjs';
/**
 *
 */
export default class TextTrackCueList extends Array<TextTrackCue> {
    /**
     * Constructor.
     *
     * @param [illegalConstructorSymbol] Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol?: symbol);
    /**
     * Returns the first TextTrackCue object with the identifier passed to it.
     *
     * @param id Text track cue identifier.
     */
    getCueById(id: string): TextTrackCue | null;
}
//# sourceMappingURL=TextTrackCueList.d.ts.map