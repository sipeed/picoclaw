import BrowserWindow from '../../window/BrowserWindow.js';
import Response from '../Response.js';
import ISyncResponse from '../types/ISyncResponse.js';
/**
 * Virtual server utility.
 */
export default class VirtualServerUtility {
    /**
     * Returns the filesystem path for a request URL if it matches a virtual server.
     *
     * @param window Window.
     * @param requestURL Request URL.
     */
    static getFilepath(window: BrowserWindow, requestURL: string): string | null;
    /**
     * Returns a 404 response.
     *
     * @param window Window.
     * @returns 404 response.
     */
    static getNotFoundResponse(window: BrowserWindow): Response;
    /**
     * Returns a 404 response.
     *
     * @param window Window.
     * @returns 404 response.
     */
    static getNotFoundSyncResponse(window: BrowserWindow): ISyncResponse;
}
//# sourceMappingURL=VirtualServerUtility.d.ts.map