import MediaStreamTrack from '../html-media-element/MediaStreamTrack.js';
import HTMLCanvasElement from './HTMLCanvasElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * Canvas Capture Media Stream Track.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/CanvasCaptureMediaStreamTrack
 */
export default class CanvasCaptureMediaStreamTrack extends MediaStreamTrack {
	public [PropertySymbol.canvas]: HTMLCanvasElement;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param canvas Canvas.
	 */
	constructor(illegalConstructorSymbol: symbol, canvas: HTMLCanvasElement) {
		super(illegalConstructorSymbol);

		this[PropertySymbol.canvas] = canvas;
	}

	/**
	 * Returns the canvas.
	 *
	 * @returns Canvas.
	 */
	public get canvas(): HTMLCanvasElement {
		return this[PropertySymbol.canvas];
	}

	/**
	 * Requests a frame.
	 */
	public requestFrame(): void {
		// Do nothing
	}

	/**
	 * Clones the track.
	 *
	 * @returns Clone.
	 */
	public clone(): CanvasCaptureMediaStreamTrack {
		const clone = <CanvasCaptureMediaStreamTrack>super.clone();
		clone[PropertySymbol.canvas] = this.canvas;
		return clone;
	}
}
