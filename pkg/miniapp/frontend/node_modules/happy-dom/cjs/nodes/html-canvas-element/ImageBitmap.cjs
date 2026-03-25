"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Image Bitmap.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/ImageBitmap
 */
class ImageBitmap {
    height;
    width;
    /**
     * Constructor.
     *
     * @param width Width.
     * @param height Height.
     */
    constructor(width, height) {
        this.width = width;
        this.height = height;
    }
    /**
     * Disposes of all graphical resources associated with an ImageBitmap.
     */
    close() {
        // TODO: Not implemented.
    }
}
exports.default = ImageBitmap;
//# sourceMappingURL=ImageBitmap.cjs.map