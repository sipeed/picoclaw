"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const url_1 = require("url");
/**
 * Browser frame URL utility.
 */
class BrowserFrameURL {
    /**
     * Returns relative URL.
     *
     * @param frame Frame.
     * @param url URL.
     * @returns Relative URL.
     */
    static getRelativeURL(frame, url) {
        url = url ? String(url) : 'about:blank';
        if (url.startsWith('about:') || url.startsWith('javascript:')) {
            return new url_1.URL(url);
        }
        try {
            return new url_1.URL(url, frame.window.location.href);
        }
        catch (e) {
            return new url_1.URL('about:blank');
        }
    }
}
exports.default = BrowserFrameURL;
//# sourceMappingURL=BrowserFrameURL.cjs.map