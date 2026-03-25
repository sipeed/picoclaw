"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DefaultCookie_js_1 = __importDefault(require("./DefaultCookie.cjs"));
const CookieExpireUtility_js_1 = __importDefault(require("./urilities/CookieExpireUtility.cjs"));
const CookieURLUtility_js_1 = __importDefault(require("./urilities/CookieURLUtility.cjs"));
/**
 * Cookie Container.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cookie.
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie.
 */
class CookieContainer {
    #cookies = [];
    /**
     * Adds cookies.
     *
     * @param cookies Cookies.
     */
    addCookies(cookies) {
        const indexMap = {};
        const getKey = (cookie) => `${cookie.key}-${cookie.originURL.hostname}-${cookie.path}-${typeof cookie.value}`;
        // Creates a map of cookie key, domain, path and value to index.
        for (let i = 0, max = this.#cookies.length; i < max; i++) {
            indexMap[getKey(this.#cookies[i])] = i;
        }
        for (const cookie of cookies) {
            const newCookie = Object.assign({}, DefaultCookie_js_1.default, cookie);
            if (newCookie && newCookie.key && newCookie.originURL) {
                // Remove existing cookie with same name, domain and path.
                const index = indexMap[getKey(newCookie)];
                if (index !== undefined) {
                    this.#cookies.splice(index, 1);
                }
                if (!CookieExpireUtility_js_1.default.hasExpired(newCookie)) {
                    indexMap[getKey(newCookie)] = this.#cookies.length;
                    this.#cookies.push(newCookie);
                }
            }
        }
    }
    /**
     * Returns cookies.
     *
     * @param [url] URL.
     * @param [httpOnly] "true" if only http cookies should be returned.
     * @returns Cookies.
     */
    getCookies(url = null, httpOnly = false) {
        const cookies = [];
        for (const cookie of this.#cookies) {
            if (!CookieExpireUtility_js_1.default.hasExpired(cookie) &&
                (!httpOnly || !cookie.httpOnly) &&
                (!url || CookieURLUtility_js_1.default.cookieMatchesURL(cookie, url || cookie.originURL))) {
                cookies.push(cookie);
            }
        }
        return cookies;
    }
}
exports.default = CookieContainer;
//# sourceMappingURL=CookieContainer.cjs.map