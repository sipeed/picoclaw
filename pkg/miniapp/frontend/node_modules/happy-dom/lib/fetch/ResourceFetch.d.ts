import BrowserWindow from '../window/BrowserWindow.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
/**
 * Helper class for performing fetch of resources.
 */
export default class ResourceFetch {
    #private;
    private window;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.browserFrame Browser frame.
     * @param options.window Window.
     */
    constructor(options: {
        browserFrame: IBrowserFrame;
        window: BrowserWindow;
    });
    /**
     * Returns resource data asynchronously.
     *
     * @param url URL.
     * @returns Response.
     */
    fetch(url: string): Promise<string>;
    /**
     * Returns resource data synchronously.
     *
     * @param url URL.
     * @returns Response.
     */
    fetchSync(url: string): string;
}
//# sourceMappingURL=ResourceFetch.d.ts.map