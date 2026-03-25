import FormData from '../../form-data/FormData.js';
import { ReadableStream } from 'stream/web';
import { Buffer } from 'buffer';
import BrowserWindow from '../../window/BrowserWindow.js';
/**
 * Multipart form data factory.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/utils/multipart-parser.js (MIT)
 */
export default class MultipartFormDataParser {
    /**
     * Returns form data.
     *
     * @param window Window.
     * @param body Body.
     * @param contentType Content type header value.
     * @returns Form data.
     */
    static streamToFormData(window: BrowserWindow, body: ReadableStream, contentType: string): Promise<{
        formData: FormData;
        buffer: Buffer;
    }>;
    /**
     * Converts a FormData object to a ReadableStream.
     *
     * @param formData FormData.
     * @returns Stream and type.
     */
    static formDataToStream(formData: FormData): {
        contentType: string;
        contentLength: number;
        buffer: Buffer;
        stream: ReadableStream;
    };
    /**
     * Escapes a form data entry name.
     *
     * @param name Name.
     * @param filename Whether it is a filename.
     * @returns Escaped name.
     */
    private static escapeName;
}
//# sourceMappingURL=MultipartFormDataParser.d.ts.map