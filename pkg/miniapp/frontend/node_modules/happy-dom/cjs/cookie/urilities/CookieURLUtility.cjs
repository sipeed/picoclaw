"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CookieSameSiteEnum_js_1 = __importDefault(require("../enums/CookieSameSiteEnum.cjs"));
/**
 * Cookie string.
 */
class CookieURLUtility {
    /**
     * Returns "true" if cookie matches URL.
     *
     * @param cookie Cookie.
     * @param url URL.
     * @returns "true" if cookie matches URL.
     */
    static cookieMatchesURL(cookie, url) {
        return ((!cookie.secure || url.protocol === 'https:') &&
            (!cookie.domain || url.hostname.endsWith(cookie.domain)) &&
            (!cookie.path || url.pathname.startsWith(cookie.path)) &&
            // @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie#samesitesamesite-value
            ((cookie.sameSite === CookieSameSiteEnum_js_1.default.none && cookie.secure) ||
                cookie.originURL.hostname === url.hostname));
    }
}
exports.default = CookieURLUtility;
//# sourceMappingURL=CookieURLUtility.cjs.map