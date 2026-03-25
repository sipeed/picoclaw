import URL from '../url/URL.js';
import DefaultCookie from './DefaultCookie.js';
import ICookie from './ICookie.js';
import ICookieContainer from './ICookieContainer.js';
import IOptionalCookie from './IOptionalCookie.js';
import CookieExpireUtility from './urilities/CookieExpireUtility.js';
import CookieURLUtility from './urilities/CookieURLUtility.js';

/**
 * Cookie Container.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cookie.
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie.
 */
export default class CookieContainer implements ICookieContainer {
	#cookies: ICookie[] = [];

	/**
	 * Adds cookies.
	 *
	 * @param cookies Cookies.
	 */
	public addCookies(cookies: IOptionalCookie[]): void {
		const indexMap: { [k: string]: number } = {};
		const getKey = (cookie: ICookie): string =>
			`${cookie.key}-${cookie.originURL.hostname}-${cookie.path}-${typeof cookie.value}`;

		// Creates a map of cookie key, domain, path and value to index.
		for (let i = 0, max = this.#cookies.length; i < max; i++) {
			indexMap[getKey(this.#cookies[i])] = i;
		}

		for (const cookie of cookies) {
			const newCookie = Object.assign({}, DefaultCookie, cookie);

			if (newCookie && newCookie.key && newCookie.originURL) {
				// Remove existing cookie with same name, domain and path.
				const index = indexMap[getKey(newCookie)];

				if (index !== undefined) {
					this.#cookies.splice(index, 1);
				}

				if (!CookieExpireUtility.hasExpired(newCookie)) {
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
	public getCookies(url: URL | null = null, httpOnly = false): ICookie[] {
		const cookies = [];

		for (const cookie of this.#cookies) {
			if (
				!CookieExpireUtility.hasExpired(cookie) &&
				(!httpOnly || !cookie.httpOnly) &&
				(!url || CookieURLUtility.cookieMatchesURL(cookie, url || cookie.originURL))
			) {
				cookies.push(cookie);
			}
		}

		return cookies;
	}
}
