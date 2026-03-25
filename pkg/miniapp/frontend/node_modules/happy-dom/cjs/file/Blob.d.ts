import { Buffer } from 'buffer';
import { ReadableStream } from 'stream/web';
import * as PropertySymbol from '../PropertySymbol.cjs';
/**
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Blob.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/Blob-impl.js (MIT licensed).
 */
export default class Blob {
    readonly type: string;
    [PropertySymbol.buffer]: Buffer;
    /**
     * Constructor.
     *
     * @param [bits] Bits.
     * @param [options] Options.
     * @param [options.type] MIME type.
     */
    constructor(bits?: (ArrayBuffer | ArrayBufferView | Blob | Buffer | string)[], options?: {
        type?: string;
    });
    /**
     * Returns size.
     *
     * @returns Size.
     */
    get size(): number;
    /**
     * Slices the blob.
     *
     * @param start Start.
     * @param end End.
     * @param contentType Content type.
     * @returns New Blob.
     */
    slice(start?: number, end?: number, contentType?: string): Blob;
    /**
     * Returns a Promise that resolves to a ArrayBuffer.
     *
     * @returns ArrayBuffer.
     */
    /**
     *
     */
    arrayBuffer(): Promise<ArrayBuffer>;
    /**
     * Returns a Promise that resolves to a text.
     *
     * @returns Text.
     */
    text(): Promise<string>;
    /**
     * Returns returns a ReadableStream which upon reading returns the data contained within the Blob.
     *
     * @returns ReadableStream
     */
    stream(): ReadableStream;
    /**
     * Returns the object converted to string.
     *
     * @returns String.
     */
    toString(): string;
}
//# sourceMappingURL=Blob.d.ts.map