import TextTrackCue from './TextTrackCue.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * VTTCue.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/VTTCue
 */
export default class VTTCue extends TextTrackCue {
    region = null;
    vertical = '';
    snapToLines = true;
    line = 0;
    lineAlign = '';
    position = 'auto';
    positionAlign = 'auto';
    size = 100;
    align = '';
    text = '';
    /**
     * Constructor.
     *
     * @param startTime The start time for the cue.
     * @param endTime The end time for the cue.
     * @param text The text of the cue.
     */
    constructor(startTime, endTime, text) {
        super(PropertySymbol.illegalConstructor);
        const window = this[PropertySymbol.window];
        // TODO: Can we find a better solution for counting arguments by using the "arguments" property?
        let argumentCount = 0;
        if (startTime !== undefined) {
            argumentCount++;
        }
        if (endTime !== undefined) {
            argumentCount++;
        }
        if (text !== undefined) {
            argumentCount++;
        }
        if (argumentCount < 3) {
            throw new window.TypeError(`Failed to construct 'VTTCue': 3 arguments required, but only ${argumentCount} present.`);
        }
        startTime = Number(startTime);
        endTime = Number(endTime);
        if (isNaN(startTime) || isNaN(endTime)) {
            throw new window.TypeError(`Failed to construct 'VTTCue': The provided double value is non-finite.`);
        }
        this.startTime = startTime;
        this.endTime = endTime;
        this.text = String(text);
    }
    /**
     * Returns the cue as HTML.
     *
     * @returns DocumentFragment
     */
    getCueAsHTML() {
        const window = this[PropertySymbol.window];
        const fragment = window.document.createDocumentFragment();
        fragment.appendChild(window.document.createTextNode(this.text));
        return fragment;
    }
}
//# sourceMappingURL=VTTCue.js.map