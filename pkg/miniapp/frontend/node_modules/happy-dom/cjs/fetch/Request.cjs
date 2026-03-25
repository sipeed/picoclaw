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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const URL_js_1 = __importDefault(require("../url/URL.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const Headers_js_1 = __importDefault(require("./Headers.cjs"));
const FetchBodyUtility_js_1 = __importDefault(require("./utilities/FetchBodyUtility.cjs"));
const Blob_js_1 = __importDefault(require("../file/Blob.cjs"));
const FetchRequestValidationUtility_js_1 = __importDefault(require("./utilities/FetchRequestValidationUtility.cjs"));
const FetchRequestReferrerUtility_js_1 = __importDefault(require("./utilities/FetchRequestReferrerUtility.cjs"));
const FetchRequestHeaderUtility_js_1 = __importDefault(require("./utilities/FetchRequestHeaderUtility.cjs"));
const MultipartFormDataParser_js_1 = __importDefault(require("./multipart/MultipartFormDataParser.cjs"));
const WindowBrowserContext_js_1 = __importDefault(require("../window/WindowBrowserContext.cjs"));
/**
 * Fetch request.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/request.js
 *
 * @see https://fetch.spec.whatwg.org/#request-class
 */
class Request {
    // Public properties
    method;
    body;
    headers;
    redirect;
    referrerPolicy;
    signal;
    bodyUsed = false;
    credentials;
    // Internal properties
    [PropertySymbol.aborted] = false;
    [PropertySymbol.contentLength] = null;
    [PropertySymbol.contentType] = null;
    [PropertySymbol.referrer] = 'client';
    [PropertySymbol.url];
    [PropertySymbol.bodyBuffer];
    /**
     * Constructor.
     *
     * @param input Input.
     * @param [init] Init.
     */
    constructor(input, init) {
        const window = this[PropertySymbol.window];
        if (!window) {
            throw new TypeError(`Failed to construct 'Request': 'Request' was constructed outside a Window context.`);
        }
        if (!input) {
            throw new window.TypeError(`Failed to contruct 'Request': 1 argument required, only 0 present.`);
        }
        this.method = (init?.method || input.method || 'GET').toUpperCase();
        const { stream, buffer, contentType, contentLength } = FetchBodyUtility_js_1.default.getBodyStream(input instanceof Request && (input[PropertySymbol.bodyBuffer] || input.body)
            ? input[PropertySymbol.bodyBuffer] || FetchBodyUtility_js_1.default.cloneBodyStream(window, input)
            : init?.body);
        this[PropertySymbol.bodyBuffer] = buffer;
        this.body = stream;
        this.credentials = init?.credentials || input.credentials || 'same-origin';
        this.headers = new Headers_js_1.default(init?.headers || input.headers || {});
        FetchRequestHeaderUtility_js_1.default.removeForbiddenHeaders(this.headers);
        if (contentLength) {
            this[PropertySymbol.contentLength] = contentLength;
        }
        else if (!this.body && (this.method === 'POST' || this.method === 'PUT')) {
            this[PropertySymbol.contentLength] = 0;
        }
        if (contentType) {
            if (!this.headers.has('Content-Type')) {
                this.headers.set('Content-Type', contentType);
            }
            this[PropertySymbol.contentType] = contentType;
        }
        else if (input instanceof Request && input[PropertySymbol.contentType]) {
            this[PropertySymbol.contentType] = input[PropertySymbol.contentType];
        }
        this.redirect = init?.redirect || input.redirect || 'follow';
        this.referrerPolicy = ((init?.referrerPolicy || input.referrerPolicy || '').toLowerCase());
        this.signal = init?.signal || input.signal || new window.AbortSignal();
        this[PropertySymbol.referrer] = FetchRequestReferrerUtility_js_1.default.getInitialReferrer(window, init?.referrer !== null && init?.referrer !== undefined
            ? init?.referrer
            : input.referrer);
        if (input instanceof URL_js_1.default) {
            this[PropertySymbol.url] = input;
        }
        else {
            try {
                if (input instanceof Request && input.url) {
                    this[PropertySymbol.url] = new URL_js_1.default(input.url, window.location.href);
                }
                else {
                    this[PropertySymbol.url] = new URL_js_1.default(input, window.location.href);
                }
            }
            catch (error) {
                throw new window.DOMException(`Failed to construct 'Request': Invalid URL "${input}" on document location '${window.location}'.${window.location.origin === 'null'
                    ? ' Relative URLs are not permitted on current document location.'
                    : ''}`, DOMExceptionNameEnum_js_1.default.notSupportedError);
            }
        }
        FetchRequestValidationUtility_js_1.default.validateMethod(this);
        FetchRequestValidationUtility_js_1.default.validateBody(this);
        FetchRequestValidationUtility_js_1.default.validateURL(this[PropertySymbol.url]);
        FetchRequestValidationUtility_js_1.default.validateReferrerPolicy(this.referrerPolicy);
        FetchRequestValidationUtility_js_1.default.validateRedirect(this.redirect);
    }
    /**
     * Returns referrer.
     *
     * @returns Referrer.
     */
    get referrer() {
        if (!this[PropertySymbol.referrer] || this[PropertySymbol.referrer] === 'no-referrer') {
            return '';
        }
        if (this[PropertySymbol.referrer] === 'client') {
            return 'about:client';
        }
        return this[PropertySymbol.referrer].toString();
    }
    /**
     * Returns URL.
     *
     * @returns URL.
     */
    get url() {
        return this[PropertySymbol.url].href;
    }
    /**
     * Returns string tag.
     *
     * @returns String tag.
     */
    get [Symbol.toStringTag]() {
        return 'Request';
    }
    /**
     * Returns array buffer.
     *
     * @returns Array buffer.
     */
    async arrayBuffer() {
        const window = this[PropertySymbol.window];
        if (this.bodyUsed) {
            throw new window.DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const asyncTaskManager = new WindowBrowserContext_js_1.default(window).getAsyncTaskManager();
        this.bodyUsed = true;
        const taskID = asyncTaskManager.startTask(() => {
            if (this.body) {
                this.body[PropertySymbol.aborted] = true;
            }
            this.signal[PropertySymbol.abort]();
        });
        let buffer;
        try {
            buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(window, this.body);
        }
        catch (error) {
            asyncTaskManager.endTask(taskID);
            throw error;
        }
        asyncTaskManager.endTask(taskID);
        return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
    }
    /**
     * Returns blob.
     *
     * @returns Blob.
     */
    async blob() {
        const type = this.headers.get('Content-Type') || '';
        const buffer = await this.arrayBuffer();
        return new Blob_js_1.default([buffer], { type });
    }
    /**
     * Returns buffer.
     *
     * @returns Buffer.
     */
    async buffer() {
        const window = this[PropertySymbol.window];
        if (this.bodyUsed) {
            throw new window.DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const asyncTaskManager = new WindowBrowserContext_js_1.default(window).getAsyncTaskManager();
        this.bodyUsed = true;
        const taskID = asyncTaskManager.startTask(() => {
            if (this.body) {
                this.body[PropertySymbol.aborted] = true;
            }
            this.signal[PropertySymbol.abort]();
        });
        let buffer;
        try {
            buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(window, this.body);
        }
        catch (error) {
            asyncTaskManager.endTask(taskID);
            throw error;
        }
        asyncTaskManager.endTask(taskID);
        return buffer;
    }
    /**
     * Returns text.
     *
     * @returns Text.
     */
    async text() {
        const window = this[PropertySymbol.window];
        if (this.bodyUsed) {
            throw new window.DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const asyncTaskManager = new WindowBrowserContext_js_1.default(window).getAsyncTaskManager();
        this.bodyUsed = true;
        const taskID = asyncTaskManager.startTask(() => {
            if (this.body) {
                this.body[PropertySymbol.aborted] = true;
            }
            this.signal[PropertySymbol.abort]();
        });
        let buffer;
        try {
            buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(window, this.body);
        }
        catch (error) {
            asyncTaskManager.endTask(taskID);
            throw error;
        }
        asyncTaskManager.endTask(taskID);
        return new TextDecoder().decode(buffer);
    }
    /**
     * Returns json.
     *
     * @returns JSON.
     */
    async json() {
        const text = await this.text();
        return JSON.parse(text);
    }
    /**
     * Returns FormData.
     *
     * @returns FormData.
     */
    async formData() {
        const window = this[PropertySymbol.window];
        const asyncTaskManager = new WindowBrowserContext_js_1.default(window).getAsyncTaskManager();
        const contentType = this[PropertySymbol.contentType];
        if (/multipart/i.test(contentType)) {
            if (this.bodyUsed) {
                throw new window.DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
            }
            this.bodyUsed = true;
            const taskID = asyncTaskManager.startTask(() => {
                if (this.body) {
                    this.body[PropertySymbol.aborted] = true;
                }
                this.signal[PropertySymbol.abort]();
            });
            let formData;
            try {
                const result = await MultipartFormDataParser_js_1.default.streamToFormData(window, this.body, contentType);
                formData = result.formData;
            }
            catch (error) {
                asyncTaskManager.endTask(taskID);
                throw error;
            }
            asyncTaskManager.endTask(taskID);
            return formData;
        }
        if (contentType?.startsWith('application/x-www-form-urlencoded')) {
            const parameters = new URLSearchParams(await this.text());
            const formData = new window.FormData();
            for (const [key, value] of parameters) {
                formData.append(key, value);
            }
            return formData;
        }
        throw new window.DOMException(`Failed to construct FormData object: The "content-type" header is neither "application/x-www-form-urlencoded" nor "multipart/form-data".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
    }
    /**
     * Clones request.
     *
     * @returns Clone.
     */
    clone() {
        return new this[PropertySymbol.window].Request(this);
    }
}
exports.default = Request;
//# sourceMappingURL=Request.cjs.map