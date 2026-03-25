import ICookieContainer from '../../cookie/ICookieContainer.cjs';
import IResponseCache from '../../fetch/cache/response/IResponseCache.cjs';
import IBrowser from './IBrowser.cjs';
import IBrowserPage from './IBrowserPage.cjs';
import IPreflightResponseCache from '../../fetch/cache/preflight/IPreflightResponseCache.cjs';
/**
 * Browser context.
 */
export default interface IBrowserContext {
    readonly pages: IBrowserPage[];
    readonly browser: IBrowser;
    readonly cookieContainer: ICookieContainer;
    readonly responseCache: IResponseCache;
    readonly preflightResponseCache: IPreflightResponseCache;
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
    newPage(): IBrowserPage;
}
//# sourceMappingURL=IBrowserContext.d.ts.map