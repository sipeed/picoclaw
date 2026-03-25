import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import Headers from '../Headers.js';
import Request from '../Request.js';
/**
 * Fetch request header utility.
 */
export default class FetchRequestHeaderUtility {
    /**
     * Validates request headers.
     *
     * @param headers Headers.
     */
    static removeForbiddenHeaders(headers: Headers): void;
    /**
     * Returns "true" if the header is forbidden.
     *
     * @param name Header name.
     * @returns "true" if the header is forbidden.
     */
    static isHeaderForbidden(name: string): boolean;
    /**
     * Returns request headers.
     *
     * @param options Options.
     * @param options.browserFrame Browser frame.
     * @param options.window Window.
     * @param options.request Request.
     * @param [options.baseHeaders] Any base headers (may be overwritten by browser/window headers).
     * @returns Headers.
     */
    static getRequestHeaders(options: {
        browserFrame: IBrowserFrame;
        window: BrowserWindow;
        request: Request;
        baseHeaders?: Headers;
    }): {
        [key: string]: string;
    };
}
//# sourceMappingURL=FetchRequestHeaderUtility.d.ts.map