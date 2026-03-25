import VirtualConsolePrinter from '../console/VirtualConsolePrinter.cjs';
import IOptionalBrowserPageViewport from '../browser/types/IOptionalBrowserPageViewport.cjs';
import IBrowserFrame from '../browser/types/IBrowserFrame.cjs';
import IBrowserSettings from '../browser/types/IBrowserSettings.cjs';
/**
 * API for detached windows to be able to access features of the browser.
 */
export default class DetachedWindowAPI {
    #private;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame: IBrowserFrame);
    /**
     * Returns settings.
     *
     * @returns Settings.
     */
    get settings(): IBrowserSettings;
    /**
     * Returns virtual console printer.
     *
     * @returns Virtual console printer.
     */
    get virtualConsolePrinter(): VirtualConsolePrinter;
    /**
     * Waits for all async tasks to complete.
     *
     * @returns Promise.
     */
    waitUntilComplete(): Promise<void>;
    /**
     * Waits for all async tasks to complete.
     *
     * @deprecated Use waitUntilComplete() instead.
     * @returns Promise.
     */
    whenAsyncComplete(): Promise<void>;
    /**
     * Aborts all async tasks.
     */
    abort(): Promise<void>;
    /**
     * Aborts all async tasks.
     *
     * @deprecated Use abort() instead.
     */
    cancelAsync(): Promise<void>;
    /**
     * Aborts all async tasks and closes the window.
     */
    close(): Promise<void>;
    /**
     * Sets the URL without navigating the browser.
     *
     * @param url URL.
     */
    setURL(url: string): void;
    /**
     * Sets the viewport.
     *
     * @param viewport Viewport.
     */
    setViewport(viewport: IOptionalBrowserPageViewport): void;
    /**
     * Sets the window size.
     *
     * @deprecated Use setViewport() instead.
     * @param options Options.
     * @param options.width Width.
     * @param options.height Height.
     */
    setWindowSize(options: {
        width?: number;
        height?: number;
    }): void;
    /**
     * Sets the window width.
     *
     * @deprecated Use setViewport() instead.
     * @param width Width.
     */
    setInnerWidth(width: number): void;
    /**
     * Sets the window height.
     *
     * @deprecated Use setViewport() instead.
     * @param height Height.
     */
    setInnerHeight(height: number): void;
}
//# sourceMappingURL=DetachedWindowAPI.d.ts.map