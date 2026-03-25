import Blob from '../../file/Blob.js';
import ImageBitmap from './ImageBitmap.js';
/**
 * OffscreenCanvas.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/OffscreenCanvas/OffscreenCanvas
 */
export default class OffscreenCanvas {
    width;
    height;
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
     * Returns context.
     *
     * @param _contextType Context type.
     * @param [_contextAttributes] Context attributes.
     * @returns Context.
     */
    getContext(_contextType, _contextAttributes) {
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
    async convertToBlob(_options) {
        return new Blob([]);
    }
    /**
     * Creates an ImageBitmap object from the most recently rendered image of the OffscreenCanvas.
     *
     * @returns ImageBitmap.
     */
    transferToImageBitmap() {
        return new ImageBitmap(this.width, this.height);
    }
}
//# sourceMappingURL=OffscreenCanvas.js.map