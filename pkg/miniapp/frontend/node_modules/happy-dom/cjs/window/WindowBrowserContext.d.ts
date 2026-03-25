import AsyncTaskManager from '../async-task-manager/AsyncTaskManager.cjs';
import IBrowser from '../browser/types/IBrowser.cjs';
import IBrowserContext from '../browser/types/IBrowserContext.cjs';
import IBrowserFrame from '../browser/types/IBrowserFrame.cjs';
import IBrowserPage from '../browser/types/IBrowserPage.cjs';
import IBrowserSettings from '../browser/types/IBrowserSettings.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from './BrowserWindow.cjs';
/**
 * API for accessing the Browser in a Window context without exposing the Browser as accessible properties.
 *
 * The Browser should never be exposed to scripts, as the scripts could then manipulate it, which would lead to security issues.
 */
export default class WindowBrowserContext {
    #private;
    private static [PropertySymbol.browserFrames];
    private static [PropertySymbol.windowInternalId];
    /**
     * Browser window.
     *
     * @param window Window.
     */
    constructor(window: BrowserWindow);
    /**
     * Returns the browser settings of the window.
     *
     * @returns Browser settings.
     */
    getSettings(): IBrowserSettings | null;
    /**
     * Returns the browser.
     *
     * @returns Browser.
     */
    getBrowser(): IBrowser | null;
    /**
     * Returns the browser page.
     *
     * @returns Browser page.
     */
    getBrowserPage(): IBrowserPage | null;
    /**
     * Returns the browser context.
     *
     * @returns Browser context.
     */
    getBrowserContext(): IBrowserContext | null;
    /**
     * Returns the browser frame of the window.
     *
     * @returns Browser frame.
     */
    getBrowserFrame(): IBrowserFrame | null;
    /**
     * Returns the async task manager of the window.
     *
     * @returns Async task manager.
     */
    getAsyncTaskManager(): AsyncTaskManager | null;
    /**
     * Assigns the window to a browser frame.
     *
     * @param window Window.
     * @param browserFrame Browser frame.
     */
    static setWindowBrowserFrameRelation(window: BrowserWindow, browserFrame: IBrowserFrame): void;
    /**
     * Assigns the window to a browser frame.
     *
     * @param window Window.
     * @param browserFrame Browser frame.
     */
    static removeWindowBrowserFrameRelation(window: BrowserWindow): void;
}
//# sourceMappingURL=WindowBrowserContext.d.ts.map