import DetachedBrowser from './DetachedBrowser.js';
import DetachedBrowserPage from './DetachedBrowserPage.js';
import IBrowserContext from '../types/IBrowserContext.js';
import ICookieContainer from '../../cookie/ICookieContainer.js';
import IResponseCache from '../../fetch/cache/response/IResponseCache.js';
import IPreflightResponseCache from '../../fetch/cache/preflight/IPreflightResponseCache.js';
/**
 * Detached browser context used when constructing a Window instance without a browser.
 */
export default class DetachedBrowserContext implements IBrowserContext {
    readonly pages: DetachedBrowserPage[];
    readonly browser: DetachedBrowser;
    readonly cookieContainer: ICookieContainer;
    readonly responseCache: IResponseCache;
    readonly preflightResponseCache: IPreflightResponseCache;
    /**
     * Constructor.
     *
     * @param browser Browser.
     */
    constructor(browser: DetachedBrowser);
    /**
     * Aborts all ongoing operations and destroys the context.
     */
    close(): Promise<void>;
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     *
     * @returns Promise.
     */
    waitUntilComplete(): Promise<void>;
    /**
     * Aborts all ongoing operations.
     */
    abort(): Promise<void>;
    /**
     * Creates a new page.
     *
     * @param [opener] Opener.
     * @returns Page.
     */
    newPage(): DetachedBrowserPage;
}
//# sourceMappingURL=DetachedBrowserContext.d.ts.map