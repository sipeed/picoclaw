import URL from '../../url/URL.js';
import ICookie from '../ICookie.js';
/**
 * Cookie string.
 */
export default class CookieStringUtility {
    /**
     * Returns cookie.
     *
     * @param originURL Origin URL.
     * @param cookieString Cookie string.
     * @returns Cookie.
     */
    static stringToCookie(originURL: URL, cookieString: string): ICookie | null;
    /**
     * Returns cookie string with key and value.
     *
     * @param cookies Cookies.
     * @returns Cookie string.
     */
    static cookiesToString(cookies: ICookie[]): string;
}
//# sourceMappingURL=CookieStringUtility.d.ts.map