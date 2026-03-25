import IBrowserFrame from '../types/IBrowserFrame.js';
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
    static getRelativeURL(frame: IBrowserFrame, url: string | URL): URL;
}
//# sourceMappingURL=BrowserFrameURL.d.ts.map