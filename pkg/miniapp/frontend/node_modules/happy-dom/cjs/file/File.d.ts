import Blob from './Blob.cjs';
import { Buffer } from 'buffer';
/**
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/File.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/File-impl.js (MIT licensed).
 */
export default class File extends Blob {
    readonly lastModified: number;
    readonly name: string;
    /**
     * Constructor.
     *
     * @param bits File bits.
     * @param name File name.
     * @param [options] Options.
     * @param [options.type] MIME type.
     * @param [options.lastModifier] Last modified. Defaults to Date.now().
     * @param options.lastModified
     */
    constructor(bits: (ArrayBuffer | ArrayBufferView | Blob | Buffer | string)[], name: string, options?: {
        type?: string;
        lastModified?: number;
    });
}
//# sourceMappingURL=File.d.ts.map