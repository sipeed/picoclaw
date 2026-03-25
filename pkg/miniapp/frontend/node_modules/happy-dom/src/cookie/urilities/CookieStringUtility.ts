import CookieSameSiteEnum from '../enums/CookieSameSiteEnum.js';
import URL from '../../url/URL.js';
import ICookie from '../ICookie.js';
import DefaultCookie from '../DefaultCookie.js';

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
	public static stringToCookie(originURL: URL, cookieString: string): ICookie | null {
		const parts = cookieString.split(';');
		const part = parts.shift();
		const index = part.indexOf('=');
		const key = index !== -1 ? part.slice(0, index).trim() : part.trim();
		const value = index !== -1 ? part.slice(index + 1).trim() : null;

		const cookie: ICookie = Object.assign({}, DefaultCookie, {
			// Required
			key,
			value,
			originURL
		});

		// Invalid if key is empty.
		if (!cookie.key) {
			return null;
		}

		for (const part of parts) {
			const index = part.indexOf('=');
			const key =
				index !== -1 ? part.slice(0, index).trim().toLowerCase() : part.trim().toLowerCase();
			const value = index !== -1 ? part.slice(index + 1).trim() : '';

			switch (key) {
				case 'expires':
					cookie.expires = new Date(value);
					break;
				case 'max-age':
					cookie.expires = new Date(parseInt(value, 10) * 1000 + Date.now());
					break;
				case 'domain':
					cookie.domain = value;
					break;
				case 'path':
					cookie.path = value[0] === '/' ? value : `/${value}`;
					break;
				case 'httponly':
					cookie.httpOnly = true;
					break;
				case 'secure':
					cookie.secure = true;
					break;
				case 'samesite':
					switch (value.toLowerCase()) {
						case 'strict':
							cookie.sameSite = CookieSameSiteEnum.strict;
							break;
						case 'lax':
							cookie.sameSite = CookieSameSiteEnum.lax;
							break;
						case 'none':
							cookie.sameSite = CookieSameSiteEnum.none;
					}
					break;
			}
		}

		const lowerKey = cookie.key.toLowerCase();

		// Invalid if __secure- prefix is used and cookie is not secure.
		if (lowerKey.startsWith('__secure-') && !cookie.secure) {
			return null;
		}

		// Invalid if __host- prefix is used and cookie is not secure, not on root path or has a domain.
		if (
			lowerKey.startsWith('__host-') &&
			(!cookie.secure || cookie.path !== '/' || cookie.domain)
		) {
			return null;
		}

		return cookie;
	}

	/**
	 * Returns cookie string with key and value.
	 *
	 * @param cookies Cookies.
	 * @returns Cookie string.
	 */
	public static cookiesToString(cookies: ICookie[]): string {
		const cookieString: string[] = [];

		for (const cookie of cookies) {
			if (cookie.value !== null) {
				cookieString.push(`${cookie.key}=${cookie.value}`);
			} else {
				cookieString.push(cookie.key);
			}
		}

		return cookieString.join('; ');
	}
}
