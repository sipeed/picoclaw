import Blob from '../../file/Blob.cjs';
import ImageBitmap from './ImageBitmap.cjs';
/**
 * OffscreenCanvas.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/OffscreenCanvas/OffscreenCanvas
 */
export default class OffscreenCanvas {
    readonly width: number;
    readonly height: number;
    /**
     * Constructor.
     *
     * @param width Width.
     * @param height Height.
     */
    constructor(width: number, height: number);
    /**
     * Returns context.
     *
     * @param _contextType Context type.
     * @param [_contextAttributes] Context attributes.
     * @returns Context.
     */
    getContext(_contextType: '2d' | 'webgl' | 'webgl2' | 'webgpu' | 'bitmaprenderer', _contextAttributes?: {
        [key: string]: any;
    }): null;
    /**
     * Converts the canvas to a Blob.
     *
     * @param [_options] Options.
     * @param [_options.type] Type.
     * @param [_options.quality] Quality.
     * @returns Blob.
     */
    convertToBlob(_options?: {
        type?: string;
        quality?: any;
    }): Promise<Blob>;
    /**
     * Creates an ImageBitmap object from the most recently rendered image of the OffscreenCanvas.
     *
     * @returns ImageBitmap.
     */
    transferToImageBitmap(): ImageBitmap;
}
//# sourceMappingURL=OffscreenCanvas.d.ts.map