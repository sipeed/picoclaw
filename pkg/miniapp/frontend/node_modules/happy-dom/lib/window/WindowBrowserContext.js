import * as PropertySymbol from '../PropertySymbol.js';
/**
 * API for accessing the Browser in a Window context without exposing the Browser as accessible properties.
 *
 * The Browser should never be exposed to scripts, as the scripts could then manipulate it, which would lead to security issues.
 */
export default class WindowBrowserContext {
    static [PropertySymbol.browserFrames] = new Map();
    static [PropertySymbol.windowInternalId] = 0;
    #window;
    /**
     * Browser window.
     *
     * @param window Window.
     */
    constructor(window) {
        this.#window = window;
    }
    /**
     * Returns the browser settings of the window.
     *
     * @returns Browser settings.
     */
    getSettings() {
        return this.getBrowserFrame()?.page?.context?.browser?.settings || null;
    }
    /**
     * Returns the browser.
     *
     * @returns Browser.
     */
    getBrowser() {
        return this.getBrowserFrame()?.page?.context?.browser || null;
    }
    /**
     * Returns the browser page.
     *
     * @returns Browser page.
     */
    getBrowserPage() {
        return this.getBrowserFrame()?.page || null;
    }
    /**
     * Returns the browser context.
     *
     * @returns Browser context.
     */
    getBrowserContext() {
        return this.getBrowserFrame()?.page?.context || null;
    }
    /**
     * Returns the browser frame of the window.
     *
     * @returns Browser frame.
     */
    getBrowserFrame() {
        if (!this.#window) {
            return null;
        }
        return (this.constructor[PropertySymbol.browserFrames].get(this.#window[PropertySymbol.internalId]) || null);
    }
    /**
     * Returns the async task manager of the window.
     *
     * @returns Async task manager.
     */
    getAsyncTaskManager() {
        return this.getBrowserFrame()?.[PropertySymbol.asyncTaskManager] || null;
    }
    /**
     * Assigns the window to a browser frame.
     *
     * @param window Window.
     * @param browserFrame Browser frame.
     */
    static setWindowBrowserFrameRelation(window, browserFrame) {
        const browserFrames = this[PropertySymbol.browserFrames];
        if (window[PropertySymbol.internalId] === -1) {
            window[PropertySymbol.internalId] = this[PropertySymbol.windowInternalId];
            this[PropertySymbol.windowInternalId]++;
        }
        browserFrames.set(window[PropertySymbol.internalId], browserFrame);
    }
    /**
     * Assigns the window to a browser frame.
     *
     * @param window Window.
     * @param browserFrame Browser frame.
     */
    static removeWindowBrowserFrameRelation(window) {
        this[PropertySymbol.browserFrames].delete(window[PropertySymbol.internalId]);
    }
}
//# sourceMappingURL=WindowBrowserContext.js.map