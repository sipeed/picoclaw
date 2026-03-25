"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const url_1 = require("url");
/**
 * Fetch CORS utility.
 */
class FetchCORSUtility {
    /**
     * Validates request headers.
     *
     * @param originURL Origin URL.
     * @param targetURL Target URL.
     */
    static isCORS(originURL, targetURL) {
        originURL = typeof originURL === 'string' ? new url_1.URL(originURL) : originURL;
        targetURL = typeof targetURL === 'string' ? new url_1.URL(targetURL) : targetURL;
        if (targetURL.protocol === 'about:' || targetURL.protocol === 'javascript:') {
            return false;
        }
        return ((originURL.hostname !== targetURL.hostname &&
            !originURL.hostname.endsWith(targetURL.hostname)) ||
            originURL.protocol !== targetURL.protocol);
    }
}
exports.default = FetchCORSUtility;
//# sourceMappingURL=FetchCORSUtility.cjs.map