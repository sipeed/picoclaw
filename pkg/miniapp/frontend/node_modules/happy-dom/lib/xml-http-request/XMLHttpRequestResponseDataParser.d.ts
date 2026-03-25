import BrowserWindow from '../window/BrowserWindow.js';
import Blob from '../file/Blob.js';
import Document from '../nodes/document/Document.js';
import { Buffer } from 'buffer';
/**
 *
 */
export default class XMLHttpRequestResponseDataParser {
    /**
     * Parses response.
     *
     * @param options Options.
     * @param options.window Window.
     * @param [options.responseType] Response type.
     * @param [options.data] Data.
     * @param [options.contentType] Content type.
     * @returns Parsed response.
     **/
    static parse(options: {
        window: BrowserWindow;
        responseType: string;
        data?: Buffer;
        contentType?: string;
    }): ArrayBuffer | Blob | Document | object | string | null;
}
//# sourceMappingURL=XMLHttpRequestResponseDataParser.d.ts.map