import Blob from '../../file/Blob.js';
import ImageBitmap from './ImageBitmap.js';

/**
 * OffscreenCanvas.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/OffscreenCanvas/OffscreenCanvas
 */
export default class OffscreenCanvas {
	public readonly width: number;
	public readonly height: number;

	/**
	 * Constructor.
	 *
	 * @param width Width.
	 * @param height Height.
	 */
	constructor(width: number, height: number) {
		this.width = width;
		this.height = height;
	}

	/**
	 * Returns context.
	 *
	 * @param _contextType Context type.
	 * @param [_contextAttributes] Context attributes.
	 * @returns Context.
	 */
	public getContext(
		_contextType: '2d' | 'webgl' | 'webgl2' | 'webgpu' | 'bitmaprenderer',
		_contextAttributes?: { [key: string]: any }
	): null {
		return null;
	}

	/**
	 * Converts the canvas to a Blob.
	 *
	 * @param [_options] Options.
	 * @param [_options.type] Type.
	 * @param [_options.quality] Quality.
	 * @returns Blob.
	 */
	public async convertToBlob(_options?: { type?: string; quality?: any }): Promise<Blob> {
		return new Blob([]);
	}

	/**
	 * Creates an ImageBitmap object from the most recently rendered image of the OffscreenCanvas.
	 *
	 * @returns ImageBitmap.
	 */
	public transferToImageBitmap(): ImageBitmap {
		return new ImageBitmap(this.width, this.height);
	}
}
