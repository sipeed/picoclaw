"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CookieStringUtility_js_1 = __importDefault(require("../../cookie/urilities/CookieStringUtility.cjs"));
const Headers_js_1 = __importDefault(require("../Headers.cjs"));
/**
 * Fetch request validation utility.
 */
class FetchResponseHeaderUtility {
    /**
     * Appends headers to response.
     *
     * @param nodeResponse HTTP request.
     * @param options
     * @param options.browserFrame
     * @param options.requestURL
     * @param options.rawHeaders
     * @returns Headers.
     */
    static parseResponseHeaders(options) {
        const headers = new Headers_js_1.default();
        let key = null;
        for (const header of options.rawHeaders) {
            if (!key) {
                key = header;
            }
            else {
                const lowerName = key.toLowerCase();
                // Handles setting cookie headers to the document.
                // "Set-Cookie" and "Set-Cookie2" are not allowed in response headers according to spec.
                if (lowerName === 'set-cookie' || lowerName === 'set-cookie2') {
                    options.browserFrame.page.context.cookieContainer.addCookies([
                        CookieStringUtility_js_1.default.stringToCookie(options.requestURL, header)
                    ]);
                }
                else {
                    headers.append(key, header);
                }
                key = null;
            }
        }
        return headers;
    }
}
exports.default = FetchResponseHeaderUtility;
//# sourceMappingURL=FetchResponseHeaderUtility.cjs.map