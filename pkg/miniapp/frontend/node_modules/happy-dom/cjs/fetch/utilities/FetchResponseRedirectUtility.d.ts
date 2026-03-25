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
    static isRedirect(statusCode: number): boolean;
    /**
     * Returns "true" if max redirects is reached.
     *
     * @param redirectCount Redirect count.
     * @returns "true" if max redirects is reached.
     */
    static isMaxRedirectsReached(redirectCount: number): boolean;
}
//# sourceMappingURL=FetchResponseRedirectUtility.d.ts.map