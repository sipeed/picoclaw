import FormData from '../../form-data/FormData.cjs';
import BrowserWindow from '../../window/BrowserWindow.cjs';
/**
 * Multipart reader.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/utils/multipart-parser.js (MIT)
 */
export default class MultipartReader {
    private formData;
    private boundary;
    private boundaryIndex;
    private state;
    private data;
    private window;
    /**
     * Constructor.
     *
     * @param window Window.
     * @param formData Form data.
     * @param boundary Boundary.
     */
    constructor(window: BrowserWindow, boundary: string);
    /**
     * Appends data.
     *
     * @param data Data.
     */
    write(data: Uint8Array): void;
    /**
     * Ends the stream.
     *
     * @returns Form data.
     */
    end(): FormData;
    /**
     * Appends data.
     *
     * @param key Key.
     * @param value value.
     * @param filename Filename.
     * @param type Type.
     */
    private appendFormData;
    /**
     * Returns content disposition.
     *
     * @param headerValue Header value.
     * @returns Content disposition.
     */
    private getContentDisposition;
}
//# sourceMappingURL=MultipartReader.d.ts.map