import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Blob from '../../file/Blob.js';
import OffscreenCanvas from './OffscreenCanvas.js';
const DEVICE_ID = 'S3F/aBCdEfGHIjKlMnOpQRStUvWxYz1234567890+1AbC2DEf2GHi3jK34le+ab12C3+1aBCdEf==';
/**
 * HTMLCanvasElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLCanvasElement
 */
export default class HTMLCanvasElement extends HTMLElement {
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
        callback(new Blob([]));
    }
    /**
     * Transfers control to offscreen.
     *
     * @returns Offscreen canvas.
     */
    transferControlToOffscreen() {
        return new OffscreenCanvas(this.width, this.height);
    }
}
//# sourceMappingURL=HTMLCanvasElement.js.map