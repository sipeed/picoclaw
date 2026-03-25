import * as PropertySymbol from '../../PropertySymbol.js';
/**
 *
 */
export default class TextTrackCueList extends Array {
    /**
     * Constructor.
     *
     * @param [illegalConstructorSymbol] Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol) {
        super();
        // "illegalConstructorSymbol" can be "1" when calling the "splice()" method
        if (illegalConstructorSymbol !== 1 &&
            illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
    }
    /**
     * Returns the first TextTrackCue object with the identifier passed to it.
     *
     * @param id Text track cue identifier.
     */
    getCueById(id) {
        for (const cue of this) {
            if (cue.id === id) {
                return cue;
            }
        }
        return null;
    }
}
//# sourceMappingURL=TextTrackCueList.js.map