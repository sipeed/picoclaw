import BrowserWindow from '../window/BrowserWindow.cjs';
import Blob from '../file/Blob.cjs';
import Document from '../nodes/document/Document.cjs';
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