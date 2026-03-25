/**
 * Cookie expire utility.
 */
export default class CookieExpireUtility {
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
//# sourceMappingURL=CookieExpireUtility.js.map