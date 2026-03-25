import XMLHttpRequestEventTarget from './XMLHttpRequestEventTarget.js';
import XMLHttpRequestReadyStateEnum from './XMLHttpRequestReadyStateEnum.js';
import Document from '../nodes/document/Document.js';
import Blob from '../file/Blob.js';
import XMLHttpRequestUpload from './XMLHttpRequestUpload.js';
import XMLHttpResponseTypeEnum from './XMLHttpResponseTypeEnum.js';
import IRequestBody from '../fetch/types/IRequestBody.js';
/**
 * XMLHttpRequest.
 *
 * Based on:
 * https://github.com/mjwwit/node-XMLHttpRequest/blob/master/lib/XMLHttpRequest.js
 */
export default class XMLHttpRequest extends XMLHttpRequestEventTarget {
    #private;
    static UNSENT: XMLHttpRequestReadyStateEnum;
    static OPENED: XMLHttpRequestReadyStateEnum;
    static HEADERS_RECEIVED: XMLHttpRequestReadyStateEnum;
    static LOADING: XMLHttpRequestReadyStateEnum;
    static DONE: XMLHttpRequestReadyStateEnum;
    upload: XMLHttpRequestUpload;
    withCredentials: boolean;
    /**
     * Constructor.
     */
    constructor();
    /**
     * Returns the status.
     *
     * @returns Status.
     */
    get status(): number;
    /**
     * Returns the status text.
     *
     * @returns Status text.
     */
    get statusText(): string;
    /**
     * Returns the response.
     *
     * @returns Response.
     */
    get response(): ArrayBuffer | Blob | Document | object | string | null;
    /**
     * Get the response text.
     *
     * @throws {DOMException} If the response type is not text or empty.
     * @returns The response text.
     */
    get responseText(): string;
    /**
     * Get the responseXML.
     *
     * @throws {DOMException} If the response type is not text or empty.
     * @returns Response XML.
     */
    get responseXML(): Document;
    /**
     * Returns the response URL.
     *
     * @returns Response URL.
     */
    get responseURL(): string;
    /**
     * Returns the ready state.
     *
     * @returns Ready state.
     */
    get readyState(): XMLHttpRequestReadyStateEnum;
    /**
     * Set response type.
     *
     * @param type Response type.
     * @throws {DOMException} If the state is not unsent or opened.
     * @throws {DOMException} If the request is synchronous.
     */
    set responseType(type: XMLHttpResponseTypeEnum | '');
    /**
     * Get response Type.
     *
     * @returns Response type.
     */
    get responseType(): XMLHttpResponseTypeEnum | '';
    /**
     * Opens the connection.
     *
     * @param method Connection method (eg GET, POST).
     * @param url URL for the connection.
     * @param [async=true] Asynchronous connection.
     * @param [user] Username for basic authentication (optional).
     * @param [password] Password for basic authentication (optional).
     */
    open(method: string, url: string, async?: boolean, user?: string, password?: string): void;
    /**
     * Sets a header for the request.
     *
     * @param name Header name.
     * @param value Header value.
     * @returns Header added.
     */
    setRequestHeader(name: string, value: string): boolean;
    /**
     * Gets a header from the server response.
     *
     * @param header header Name of header to get.
     * @returns string Text of the header or null if it doesn't exist.
     */
    getResponseHeader(header: string): string | null;
    /**
     * Gets all the response headers.
     *
     * @returns A string with all response headers separated by CR+LF.
     */
    getAllResponseHeaders(): string;
    /**
     * Sends the request to the server.
     *
     * @param body Optional data to send as request body.
     */
    send(body?: Document | IRequestBody): void;
    /**
     * Aborts a request.
     */
    abort(): void;
}
//# sourceMappingURL=XMLHttpRequest.d.ts.map