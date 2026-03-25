import { URL } from 'url';
/**
 * Browser frame URL utility.
 */
export default class BrowserFrameURL {
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
            return new URL(url);
        }
        try {
            return new URL(url, frame.window.location.href);
        }
        catch (e) {
            return new URL('about:blank');
        }
    }
}
//# sourceMappingURL=BrowserFrameURL.js.map