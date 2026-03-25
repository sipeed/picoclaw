import { ReadableStream } from 'stream/web';
import IRequestBody from '../types/IRequestBody.js';
import IResponseBody from '../types/IResponseBody.js';
import { Buffer } from 'buffer';
import Stream from 'stream';
import BrowserWindow from '../../window/BrowserWindow.js';
/**
 * Fetch body utility.
 */
export default class FetchBodyUtility {
    /**
     * Parses body and returns stream and type.
     *
     * Based on:
     * https://github.com/node-fetch/node-fetch/blob/main/src/body.js (MIT)
     *
     * @param body Body.
     * @returns Stream and type.
     */
    static getBodyStream(body: IRequestBody | IResponseBody): {
        contentType: string;
        contentLength: number | null;
        stream: ReadableStream | null;
        buffer: Buffer | null;
    };
    /**
     * Clones a request or body body stream.
     *
     * It is actually not cloning the stream.
     * It creates a pass through stream and pipes the original stream to it.
     *
     * @param window Window.
     * @param requestOrResponse Request or Response.
     * @param requestOrResponse.body Body.
     * @param requestOrResponse.bodyUsed Body used.
     * @returns New stream.
     */
    static cloneBodyStream(window: BrowserWindow, requestOrResponse: {
        body: ReadableStream | null;
        bodyUsed: boolean;
    }): ReadableStream;
    /**
     * Consume and convert an entire Body to a Buffer.
     *
     * Based on:
     * https://github.com/node-fetch/node-fetch/blob/main/src/body.js (MIT)
     *
     * @see https://fetch.spec.whatwg.org/#concept-body-consume-body
     * @param window Window.
     * @param body Body stream.
     * @returns Promise.
     */
    static consumeBodyStream(window: BrowserWindow, body: ReadableStream | null): Promise<Buffer>;
    /**
     * Wraps a given value in a browser ReadableStream.
     *
     * This method creates a ReadableStream and immediately enqueues and closes it
     * with the provided value, useful for stream API compatibility.
     *
     * @param value The value to be wrapped in a ReadableStream.
     * @returns ReadableStream
     */
    static toReadableStream(value: any): ReadableStream;
    /**
     * Wraps a Node.js stream into a browser-compatible ReadableStream.
     *
     * Enables the use of Node.js streams where browser ReadableStreams are required.
     * Handles 'data', 'end', and 'error' events from the Node.js stream.
     *
     * @param nodeStream The Node.js stream to be converted.
     * @returns ReadableStream
     */
    static nodeToWebStream(nodeStream: Stream): ReadableStream;
}
//# sourceMappingURL=FetchBodyUtility.d.ts.map