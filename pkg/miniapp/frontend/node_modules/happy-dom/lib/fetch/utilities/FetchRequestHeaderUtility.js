import * as PropertySymbol from '../../PropertySymbol.js';
import CookieStringUtility from '../../cookie/urilities/CookieStringUtility.js';
import Headers from '../Headers.js';
import FetchCORSUtility from './FetchCORSUtility.js';
import { URL } from 'url';
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
export default class FetchRequestHeaderUtility {
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
        const headers = new Headers(options.baseHeaders);
        options.request.headers.forEach((value, key) => {
            headers.set(key, value);
        });
        const originURL = new URL(options.window.location.href);
        const isCORS = FetchCORSUtility.isCORS(originURL, options.request[PropertySymbol.url]);
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
        if (options.request[PropertySymbol.referrer] instanceof URL) {
            headers.set('Referer', options.request[PropertySymbol.referrer].href);
        }
        if (options.request.credentials === 'include' ||
            (options.request.credentials === 'same-origin' && !isCORS)) {
            const cookies = options.browserFrame.page.context.cookieContainer.getCookies(originURL, false);
            if (cookies.length > 0) {
                headers.set('Cookie', CookieStringUtility.cookiesToString(cookies));
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
//# sourceMappingURL=FetchRequestHeaderUtility.js.map