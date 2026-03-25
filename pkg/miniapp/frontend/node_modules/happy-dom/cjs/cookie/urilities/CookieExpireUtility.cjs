"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Cookie expire utility.
 */
class CookieExpireUtility {
    /**
     * Returns "true" if cookie has expired.
     *
     * @param cookie Cookie.
     * @returns "true" if cookie has expired.
     */
    static hasExpired(cookie) {
        return cookie.expires && cookie.expires.getTime() < Date.now();
    }
}
exports.default = CookieExpireUtility;
//# sourceMappingURL=CookieExpireUtility.cjs.map