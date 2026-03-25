/**
 * Image Bitmap.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/ImageBitmap
 */
export default class ImageBitmap {
	public height: number;
	public width: number;

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
	 * Disposes of all graphical resources associated with an ImageBitmap.
	 */
	public close(): void {
		// TODO: Not implemented.
	}
}
