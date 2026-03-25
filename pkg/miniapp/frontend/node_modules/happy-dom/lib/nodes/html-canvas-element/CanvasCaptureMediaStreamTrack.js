import MediaStreamTrack from '../html-media-element/MediaStreamTrack.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * Canvas Capture Media Stream Track.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/CanvasCaptureMediaStreamTrack
 */
export default class CanvasCaptureMediaStreamTrack extends MediaStreamTrack {
    [PropertySymbol.canvas];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param canvas Canvas.
     */
    constructor(illegalConstructorSymbol, canvas) {
        super(illegalConstructorSymbol);
        this[PropertySymbol.canvas] = canvas;
    }
    /**
     * Returns the canvas.
     *
     * @returns Canvas.
     */
    get canvas() {
        return this[PropertySymbol.canvas];
    }
    /**
     * Requests a frame.
     */
    requestFrame() {
        // Do nothing
    }
    /**
     * Clones the track.
     *
     * @returns Clone.
     */
    clone() {
        const clone = super.clone();
        clone[PropertySymbol.canvas] = this.canvas;
        return clone;
    }
}
//# sourceMappingURL=CanvasCaptureMediaStreamTrack.js.map