"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Blob_js_1 = __importDefault(require("../../file/Blob.cjs"));
const ImageBitmap_js_1 = __importDefault(require("./ImageBitmap.cjs"));
/**
 * OffscreenCanvas.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/OffscreenCanvas/OffscreenCanvas
 */
class OffscreenCanvas {
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
        return new Blob_js_1.default([]);
    }
    /**
     * Creates an ImageBitmap object from the most recently rendered image of the OffscreenCanvas.
     *
     * @returns ImageBitmap.
     */
    transferToImageBitmap() {
        return new ImageBitmap_js_1.default(this.width, this.height);
    }
}
exports.default = OffscreenCanvas;
//# sourceMappingURL=OffscreenCanvas.cjs.map