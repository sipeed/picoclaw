import BrowserContext from './BrowserContext.js';
import BrowserSettingsFactory from './BrowserSettingsFactory.js';
import BrowserExceptionObserver from './utilities/BrowserExceptionObserver.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserErrorCaptureEnum from './enums/BrowserErrorCaptureEnum.js';
/**
 * Browser.
 *
 * Much of the interface for the browser has been taken from Puppeteer and Playwright, so that the API is familiar.
 */
export default class Browser {
    contexts;
    settings;
    console;
    [PropertySymbol.exceptionObserver] = null;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.settings] Browser settings.
     * @param [options.console] Console.
     */
    constructor(options) {
        this.console = options?.console || null;
        this.settings = BrowserSettingsFactory.createSettings(options?.settings);
        if (this.settings.errorCapture === BrowserErrorCaptureEnum.processLevel) {
            this[PropertySymbol.exceptionObserver] = new BrowserExceptionObserver();
        }
        this.contexts = [new BrowserContext(this)];
    }
    /**
     * Returns the default context.
     *
     * @returns Default context.
     */
    get defaultContext() {
        if (this.contexts.length === 0) {
            throw new Error('No default context. The browser has been closed.');
        }
        return this.contexts[0];
    }
    /**
     * Aborts all ongoing operations and destroys the browser.
     */
    async close() {
        await Promise.all(this.contexts.slice().map((context) => context.close()));
        this.contexts = [];
    }
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     *
     * @returns Promise.
     */
    async waitUntilComplete() {
        if (this.contexts.length === 0) {
            throw new Error('No default context. The browser has been closed.');
        }
        await Promise.all(this.contexts.map((page) => page.waitUntilComplete()));
    }
    /**
     * Aborts all ongoing operations.
     */
    abort() {
        // Using Promise instead of async/await to prevent microtask
        return new Promise((resolve, reject) => {
            if (!this.contexts.length) {
                resolve();
                return;
            }
            Promise.all(this.contexts.slice().map((context) => context.abort()))
                .then(() => resolve())
                .catch((error) => reject(error));
        });
    }
    /**
     * Creates a new incognito context.
     *
     * @returns Context.
     */
    newIncognitoContext() {
        if (this.contexts.length === 0) {
            throw new Error('No default context. The browser has been closed.');
        }
        const context = new BrowserContext(this);
        this.contexts.push(context);
        return context;
    }
    /**
     * Creates a new page.
     *
     * @returns Page.
     */
    newPage() {
        if (this.contexts.length === 0) {
            throw new Error('No default context. The browser has been closed.');
        }
        return this.contexts[0].newPage();
    }
}
//# sourceMappingURL=Browser.js.map