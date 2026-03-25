/**
 * Image Bitmap.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/ImageBitmap
 */
export default class ImageBitmap {
    height: number;
    width: number;
    /**
     * Constructor.
     *
     * @param width Width.
     * @param height Height.
     */
    constructor(width: number, height: number);
    /**
     * Disposes of all graphical resources associated with an ImageBitmap.
     */
    close(): void;
}
//# sourceMappingURL=ImageBitmap.d.ts.map