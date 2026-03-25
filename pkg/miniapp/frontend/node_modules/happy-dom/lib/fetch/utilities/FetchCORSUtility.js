import { URL } from 'url';
/**
 * Fetch CORS utility.
 */
export default class FetchCORSUtility {
    /**
     * Validates request headers.
     *
     * @param originURL Origin URL.
     * @param targetURL Target URL.
     */
    static isCORS(originURL, targetURL) {
        originURL = typeof originURL === 'string' ? new URL(originURL) : originURL;
        targetURL = typeof targetURL === 'string' ? new URL(targetURL) : targetURL;
        if (targetURL.protocol === 'about:' || targetURL.protocol === 'javascript:') {
            return false;
        }
        return ((originURL.hostname !== targetURL.hostname &&
            !originURL.hostname.endsWith(targetURL.hostname)) ||
            originURL.protocol !== targetURL.protocol);
    }
}
//# sourceMappingURL=FetchCORSUtility.js.map