import ICookie from '../ICookie.js';

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
	public static hasExpired(cookie: ICookie): boolean | null {
		return cookie.expires && cookie.expires.getTime() < Date.now();
	}
}
