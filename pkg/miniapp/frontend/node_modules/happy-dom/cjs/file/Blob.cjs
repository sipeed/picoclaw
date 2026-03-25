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
Object.defineProperty(exports, "__esModule", { value: true });
const buffer_1 = require("buffer");
const web_1 = require("stream/web");
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Blob.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/Blob-impl.js (MIT licensed).
 */
class Blob {
    type = '';
    [PropertySymbol.buffer] = null;
    /**
     * Constructor.
     *
     * @param [bits] Bits.
     * @param [options] Options.
     * @param [options.type] MIME type.
     */
    constructor(bits, options) {
        const buffers = [];
        if (bits) {
            for (const bit of bits) {
                let buffer;
                if (bit instanceof ArrayBuffer) {
                    buffer = buffer_1.Buffer.from(new Uint8Array(bit));
                }
                else if (bit instanceof Blob) {
                    buffer = bit[PropertySymbol.buffer];
                }
                else if (bit instanceof buffer_1.Buffer) {
                    buffer = bit;
                }
                else if (ArrayBuffer.isView(bit)) {
                    buffer = buffer_1.Buffer.from(new Uint8Array(bit.buffer, bit.byteOffset, bit.byteLength));
                }
                else {
                    buffer = buffer_1.Buffer.from(typeof bit === 'string' ? bit : String(bit));
                }
                buffers.push(buffer);
            }
        }
        this[PropertySymbol.buffer] = buffer_1.Buffer.concat(buffers);
        if (options && options.type && options.type.match(/^[\u0020-\u007E]*$/)) {
            this.type = String(options.type).toLowerCase();
        }
    }
    /**
     * Returns size.
     *
     * @returns Size.
     */
    get size() {
        return this[PropertySymbol.buffer].length;
    }
    /**
     * Slices the blob.
     *
     * @param start Start.
     * @param end End.
     * @param contentType Content type.
     * @returns New Blob.
     */
    slice(start = 0, end = null, contentType = '') {
        const size = this.size;
        let relativeStart;
        let relativeEnd;
        let relativeContentType;
        if (start === undefined) {
            relativeStart = 0;
        }
        else if (start < 0) {
            relativeStart = Math.max(size + start, 0);
        }
        else {
            relativeStart = Math.min(start, size);
        }
        if (end === null) {
            relativeEnd = size;
        }
        else if (end < 0) {
            relativeEnd = Math.max(size + end, 0);
        }
        else {
            relativeEnd = Math.min(end, size);
        }
        if (contentType === undefined) {
            relativeContentType = '';
        }
        else {
            // Sanitization (lower case and invalid char check) is done in the
            // Constructor
            relativeContentType = contentType;
        }
        const span = Math.max(relativeEnd - relativeStart, 0);
        const buffer = this[PropertySymbol.buffer];
        const slicedBuffer = buffer.slice(relativeStart, relativeStart + span);
        const blob = new Blob([], { type: relativeContentType });
        blob[PropertySymbol.buffer] = slicedBuffer;
        return blob;
    }
    /**
     * Returns a Promise that resolves to a ArrayBuffer.
     *
     * @returns ArrayBuffer.
     */
    // Reference:
    // https://github.com/web-std/io/blob/c88170bf24f064adfbb3586a21fb76650ca5a9ab/packages/blob/src/blob.js#L139-L148
    // https://stackoverflow.com/questions/8609289/convert-a-binary-nodejs-buffer-to-javascript-arraybuffer
    /**
     *
     */
    async arrayBuffer() {
        return new Uint8Array(this[PropertySymbol.buffer]).buffer;
    }
    /**
     * Returns a Promise that resolves to a text.
     *
     * @returns Text.
     */
    async text() {
        return this[PropertySymbol.buffer].toString();
    }
    /**
     * Returns returns a ReadableStream which upon reading returns the data contained within the Blob.
     *
     * @returns ReadableStream
     */
    stream() {
        const buffer = this[PropertySymbol.buffer];
        return new web_1.ReadableStream({
            start(controller) {
                controller.enqueue(buffer);
                controller.close();
            }
        });
    }
    /**
     * Returns the object converted to string.
     *
     * @returns String.
     */
    toString() {
        return '[object Blob]';
    }
}
exports.default = Blob;
//# sourceMappingURL=Blob.cjs.map