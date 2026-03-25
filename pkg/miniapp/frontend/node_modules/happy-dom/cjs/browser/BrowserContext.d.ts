import ICookieContainer from '../cookie/ICookieContainer.cjs';
import IResponseCache from '../fetch/cache/response/IResponseCache.cjs';
import Browser from './Browser.cjs';
import BrowserPage from './BrowserPage.cjs';
import IBrowserContext from './types/IBrowserContext.cjs';
import IPreflightResponseCache from '../fetch/cache/preflight/IPreflightResponseCache.cjs';
/**
 * Browser context.
 */
export default class BrowserContext implements IBrowserContext {
    readonly pages: BrowserPage[];
    readonly browser: Browser;
    readonly cookieContainer: ICookieContainer;
    readonly responseCache: IResponseCache;
    readonly preflightResponseCache: IPreflightResponseCache;
    /**
     * Constructor.
     *
     * @param browser
     */
    constructor(browser: Browser);
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
     * @returns Page.
     */
    newPage(): BrowserPage;
}
//# sourceMappingURL=BrowserContext.d.ts.map