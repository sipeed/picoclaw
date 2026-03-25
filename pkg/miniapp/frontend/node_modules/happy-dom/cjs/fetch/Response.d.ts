import * as PropertySymbol from '../PropertySymbol.cjs';
import Blob from '../file/Blob.cjs';
import IResponseInit from './types/IResponseInit.cjs';
import IResponseBody from './types/IResponseBody.cjs';
import Headers from './Headers.cjs';
import { ReadableStream } from 'stream/web';
import FormData from '../form-data/FormData.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
import ICachedResponse from './cache/response/ICachedResponse.cjs';
import { Buffer } from 'buffer';
/**
 * Fetch response.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/response.js (MIT)
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Response/Response
 */
export default class Response implements Response {
    #private;
    protected static [PropertySymbol.window]: BrowserWindow;
    protected [PropertySymbol.window]: BrowserWindow;
    readonly body: ReadableStream | null;
    readonly bodyUsed = false;
    readonly redirected = false;
    readonly type: 'basic' | 'cors' | 'default' | 'error' | 'opaque' | 'opaqueredirect';
    readonly url: string;
    readonly status: number;
    readonly statusText: string;
    readonly ok: boolean;
    readonly headers: Headers;
    [PropertySymbol.cachedResponse]: ICachedResponse | null;
    [PropertySymbol.buffer]: Buffer | null;
    /**
     * Constructor.
     *
     * @param body Body.
     * @param [init] Init.
     */
    constructor(body?: IResponseBody, init?: IResponseInit);
    /**
     * Returns string tag.
     *
     * @returns String tag.
     */
    get [Symbol.toStringTag](): string;
    /**
     * Returns array buffer.
     *
     * @returns Array buffer.
     */
    arrayBuffer(): Promise<ArrayBuffer>;
    /**
     * Returns blob.
     *
     * @returns Blob.
     */
    blob(): Promise<Blob>;
    /**
     * Returns buffer.
     *
     * @returns Buffer.
     */
    buffer(): Promise<Buffer>;
    /**
     * Returns text.
     *
     * @returns Text.
     */
    text(): Promise<string>;
    /**
     * Returns json.
     *
     * @returns JSON.
     */
    json(): Promise<string>;
    /**
     * Returns form data.
     *
     * @returns Form data.
     */
    formData(): Promise<FormData>;
    /**
     * Clones request.
     *
     * @returns Clone.
     */
    clone(): Response;
    /**
     * Returns a redirect response.
     *
     * @param url URL.
     * @param status Status code.
     * @returns Response.
     */
    static redirect(url: string, status?: number): Response;
    /**
     * Returns an error response.
     *
     * @param url URL.
     * @param status Status code.
     * @returns Response.
     */
    static error(): Response;
    /**
     * Returns an JSON response.
     *
     * @param injected Injected properties.
     * @param data Data.
     * @param [init] Init.
     * @returns Response.
     */
    static json(data: object, init?: IResponseInit): Response;
}
//# sourceMappingURL=Response.d.ts.map