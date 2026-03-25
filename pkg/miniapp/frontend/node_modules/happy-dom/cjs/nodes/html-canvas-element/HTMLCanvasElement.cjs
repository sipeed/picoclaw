"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const Blob_js_1 = __importDefault(require("../../file/Blob.cjs"));
const OffscreenCanvas_js_1 = __importDefault(require("./OffscreenCanvas.cjs"));
const DEVICE_ID = 'S3F/aBCdEfGHIjKlMnOpQRStUvWxYz1234567890+1AbC2DEf2GHi3jK34le+ab12C3+1aBCdEf==';
/**
 * HTMLCanvasElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLCanvasElement
 */
class HTMLCanvasElement extends HTMLElement_js_1.default {
    // Events
    oncontextlost = null;
    oncontextrestored = null;
    onwebglcontextcreationerror = null;
    onwebglcontextlost = null;
    onwebglcontextrestored = null;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        const width = this.getAttribute('width');
        return width !== null ? Number(width) : 300;
    }
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width) {
        this.setAttribute('width', String(width));
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        const height = this.getAttribute('height');
        return height !== null ? Number(height) : 150;
    }
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height) {
        this.setAttribute('height', String(height));
    }
    /**
     * Returns capture stream.
     *
     * @param [frameRate] Frame rate.
     * @returns Capture stream.
     */
    captureStream(frameRate) {
        const stream = new this[PropertySymbol.window].MediaStream();
        const track = new this[PropertySymbol.window].CanvasCaptureMediaStreamTrack(PropertySymbol.illegalConstructor, this);
        track[PropertySymbol.kind] = 'video';
        track[PropertySymbol.capabilities].deviceId = DEVICE_ID;
        track[PropertySymbol.capabilities].aspectRatio.max = this.width;
        track[PropertySymbol.capabilities].height.max = this.height;
        track[PropertySymbol.capabilities].width.max = this.width;
        track[PropertySymbol.settings].deviceId = DEVICE_ID;
        if (frameRate !== undefined) {
            track[PropertySymbol.capabilities].frameRate.max = frameRate;
            track[PropertySymbol.settings].frameRate = frameRate;
        }
        stream.addTrack(track);
        return stream;
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
     * Returns to data URL.
     *
     * @param [_type] Type.
     * @param [_encoderOptions] Quality.
     * @returns Data URL.
     */
    toDataURL(_type, _encoderOptions) {
        return '';
    }
    /**
     * Returns to blob.
     *
     * @param callback Callback.
     * @param [_type] Type.
     * @param [_quality] Quality.
     */
    toBlob(callback, _type, _quality) {
        callback(new Blob_js_1.default([]));
    }
    /**
     * Transfers control to offscreen.
     *
     * @returns Offscreen canvas.
     */
    transferControlToOffscreen() {
        return new OffscreenCanvas_js_1.default(this.width, this.height);
    }
}
exports.default = HTMLCanvasElement;
//# sourceMappingURL=HTMLCanvasElement.cjs.map