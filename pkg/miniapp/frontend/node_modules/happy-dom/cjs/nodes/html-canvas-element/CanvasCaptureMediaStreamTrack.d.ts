import MediaStreamTrack from '../html-media-element/MediaStreamTrack.cjs';
import HTMLCanvasElement from './HTMLCanvasElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
/**
 * Canvas Capture Media Stream Track.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/CanvasCaptureMediaStreamTrack
 */
export default class CanvasCaptureMediaStreamTrack extends MediaStreamTrack {
    [PropertySymbol.canvas]: HTMLCanvasElement;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param canvas Canvas.
     */
    constructor(illegalConstructorSymbol: symbol, canvas: HTMLCanvasElement);
    /**
     * Returns the canvas.
     *
     * @returns Canvas.
     */
    get canvas(): HTMLCanvasElement;
    /**
     * Requests a frame.
     */
    requestFrame(): void;
    /**
     * Clones the track.
     *
     * @returns Clone.
     */
    clone(): CanvasCaptureMediaStreamTrack;
}
//# sourceMappingURL=CanvasCaptureMediaStreamTrack.d.ts.map