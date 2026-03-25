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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const CookieStringUtility_js_1 = __importDefault(require("../../cookie/urilities/CookieStringUtility.cjs"));
const Headers_js_1 = __importDefault(require("../Headers.cjs"));
const FetchCORSUtility_js_1 = __importDefault(require("./FetchCORSUtility.cjs"));
const url_1 = require("url");
const FORBIDDEN_HEADER_NAMES = [
    'accept-charset',
    'accept-encoding',
    'access-control-request-headers',
    'access-control-request-method',
    'connection',
    'content-length',
    'content-transfer-encoding',
    'cookie',
    'cookie2',
    'date',
    'dnt',
    'expect',
    'host',
    'keep-alive',
    'origin',
    'referer',
    'te',
    'trailer',
    'transfer-encoding',
    'upgrade',
    'via'
];
/**
 * Fetch request header utility.
 */
class FetchRequestHeaderUtility {
    /**
     * Validates request headers.
     *
     * @param headers Headers.
     */
    static removeForbiddenHeaders(headers) {
        for (const key of Object.keys(headers[PropertySymbol.entries])) {
            if (FORBIDDEN_HEADER_NAMES.includes(key) ||
                key.startsWith('proxy-') ||
                key.startsWith('sec-')) {
                delete headers[PropertySymbol.entries][key];
            }
        }
    }
    /**
     * Returns "true" if the header is forbidden.
     *
     * @param name Header name.
     * @returns "true" if the header is forbidden.
     */
    static isHeaderForbidden(name) {
        return FORBIDDEN_HEADER_NAMES.includes(name.toLowerCase());
    }
    /**
     * Returns request headers.
     *
     * @param options Options.
     * @param options.browserFrame Browser frame.
     * @param options.window Window.
     * @param options.request Request.
     * @param [options.baseHeaders] Any base headers (may be overwritten by browser/window headers).
     * @returns Headers.
     */
    static getRequestHeaders(options) {
        const headers = new Headers_js_1.default(options.baseHeaders);
        options.request.headers.forEach((value, key) => {
            headers.set(key, value);
        });
        const originURL = new url_1.URL(options.window.location.href);
        const isCORS = FetchCORSUtility_js_1.default.isCORS(originURL, options.request[PropertySymbol.url]);
        // TODO: Maybe we need to add support for OPTIONS request with 'Access-Control-Allow-*' headers?
        if (options.request.credentials === 'omit' ||
            (options.request.credentials === 'same-origin' && isCORS)) {
            headers.delete('authorization');
            headers.delete('www-authenticate');
        }
        headers.set('Accept-Encoding', 'gzip, deflate, br');
        headers.set('Connection', 'close');
        if (!headers.has('User-Agent')) {
            headers.set('User-Agent', options.window.navigator.userAgent);
        }
        if (options.request[PropertySymbol.referrer] instanceof url_1.URL) {
            headers.set('Referer', options.request[PropertySymbol.referrer].href);
        }
        if (options.request.credentials === 'include' ||
            (options.request.credentials === 'same-origin' && !isCORS)) {
            const cookies = options.browserFrame.page.context.cookieContainer.getCookies(originURL, false);
            if (cookies.length > 0) {
                headers.set('Cookie', CookieStringUtility_js_1.default.cookiesToString(cookies));
            }
        }
        if (!headers.has('Accept')) {
            headers.set('Accept', '*/*');
        }
        if (!headers.has('Content-Length') && options.request[PropertySymbol.contentLength] !== null) {
            headers.set('Content-Length', String(options.request[PropertySymbol.contentLength]));
        }
        if (!headers.has('Content-Type') && options.request[PropertySymbol.contentType]) {
            headers.set('Content-Type', options.request[PropertySymbol.contentType]);
        }
        if (isCORS) {
            headers.set('Origin', originURL.origin);
        }
        // We need to convert the headers to Node request headers.
        const httpRequestHeaders = {};
        for (const header of Object.values(headers[PropertySymbol.entries])) {
            httpRequestHeaders[header.name] = header.value.join(', ');
        }
        return httpRequestHeaders;
    }
}
exports.default = FetchRequestHeaderUtility;
//# sourceMappingURL=FetchRequestHeaderUtility.cjs.map