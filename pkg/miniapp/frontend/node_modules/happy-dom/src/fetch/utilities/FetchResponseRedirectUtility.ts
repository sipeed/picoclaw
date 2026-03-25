const REDIRECT_STATUS_CODES = [301, 302, 303, 307, 308];
const MAX_REDIRECT_COUNT = 20;

/**
 * Fetch request validation utility.
 */
export default class FetchResponseRedirectUtility {
	/**
	 * Returns "true" if redirect.
	 *
	 * @param statusCode Status code.
	 * @returns "true" if redirect.
	 */
	public static isRedirect(statusCode: number): boolean {
		return REDIRECT_STATUS_CODES.includes(statusCode);
	}

	/**
	 * Returns "true" if max redirects is reached.
	 *
	 * @param redirectCount Redirect count.
	 * @returns "true" if max redirects is reached.
	 */
	public static isMaxRedirectsReached(redirectCount: number): boolean {
		return redirectCount >= MAX_REDIRECT_COUNT;
	}
}
