import CookieSameSiteEnum from '../enums/CookieSameSiteEnum.js';
import URL from '../../url/URL.js';
import ICookie from '../ICookie.js';

/**
 * Cookie string.
 */
export default class CookieURLUtility {
	/**
	 * Returns "true" if cookie matches URL.
	 *
	 * @param cookie Cookie.
	 * @param url URL.
	 * @returns "true" if cookie matches URL.
	 */
	public static cookieMatchesURL(cookie: ICookie, url: URL): boolean {
		return (
			(!cookie.secure || url.protocol === 'https:') &&
			(!cookie.domain || url.hostname.endsWith(cookie.domain)) &&
			(!cookie.path || url.pathname.startsWith(cookie.path)) &&
			// @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie#samesitesamesite-value
			((cookie.sameSite === CookieSameSiteEnum.none && cookie.secure) ||
				cookie.originURL.hostname === url.hostname)
		);
	}
}
