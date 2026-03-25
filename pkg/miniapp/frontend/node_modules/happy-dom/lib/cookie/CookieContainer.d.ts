import URL from '../url/URL.js';
import ICookie from './ICookie.js';
import ICookieContainer from './ICookieContainer.js';
import IOptionalCookie from './IOptionalCookie.js';
/**
 * Cookie Container.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cookie.
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie.
 */
export default class CookieContainer implements ICookieContainer {
    #private;
    /**
     * Adds cookies.
     *
     * @param cookies Cookies.
     */
    addCookies(cookies: IOptionalCookie[]): void;
    /**
     * Returns cookies.
     *
     * @param [url] URL.
     * @param [httpOnly] "true" if only http cookies should be returned.
     * @returns Cookies.
     */
    getCookies(url?: URL | null, httpOnly?: boolean): ICookie[];
}
//# sourceMappingURL=CookieContainer.d.ts.map