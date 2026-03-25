import IBrowserSettings from './types/IBrowserSettings.js';
import BrowserContext from './BrowserContext.js';
import IOptionalBrowserSettings from './types/IOptionalBrowserSettings.js';
import BrowserPage from './BrowserPage.js';
import IBrowser from './types/IBrowser.js';
import BrowserExceptionObserver from './utilities/BrowserExceptionObserver.js';
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * Browser.
 *
 * Much of the interface for the browser has been taken from Puppeteer and Playwright, so that the API is familiar.
 */
export default class Browser implements IBrowser {
    readonly contexts: BrowserContext[];
    readonly settings: IBrowserSettings;
    readonly console: Console | null;
    [PropertySymbol.exceptionObserver]: BrowserExceptionObserver | null;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.settings] Browser settings.
     * @param [options.console] Console.
     */
    constructor(options?: {
        settings?: IOptionalBrowserSettings;
        console?: Console;
    });
    /**
     * Returns the default context.
     *
     * @returns Default context.
     */
    get defaultContext(): BrowserContext;
    /**
     * Aborts all ongoing operations and destroys the browser.
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
     * Creates a new incognito context.
     *
     * @returns Context.
     */
    newIncognitoContext(): BrowserContext;
    /**
     * Creates a new page.
     *
     * @returns Page.
     */
    newPage(): BrowserPage;
}
//# sourceMappingURL=Browser.d.ts.map