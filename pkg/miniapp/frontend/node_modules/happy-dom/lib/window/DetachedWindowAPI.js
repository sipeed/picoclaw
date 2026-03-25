/**
 * API for detached windows to be able to access features of the browser.
 */
export default class DetachedWindowAPI {
    #browserFrame;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame) {
        this.#browserFrame = browserFrame;
    }
    /**
     * Returns settings.
     *
     * @returns Settings.
     */
    get settings() {
        return this.#browserFrame.page.context.browser.settings;
    }
    /**
     * Returns virtual console printer.
     *
     * @returns Virtual console printer.
     */
    get virtualConsolePrinter() {
        return this.#browserFrame.page.virtualConsolePrinter;
    }
    /**
     * Waits for all async tasks to complete.
     *
     * @returns Promise.
     */
    waitUntilComplete() {
        return this.#browserFrame.waitUntilComplete();
    }
    /**
     * Waits for all async tasks to complete.
     *
     * @deprecated Use waitUntilComplete() instead.
     * @returns Promise.
     */
    whenAsyncComplete() {
        return this.waitUntilComplete();
    }
    /**
     * Aborts all async tasks.
     */
    abort() {
        return this.#browserFrame.abort();
    }
    /**
     * Aborts all async tasks.
     *
     * @deprecated Use abort() instead.
     */
    cancelAsync() {
        return this.abort();
    }
    /**
     * Aborts all async tasks and closes the window.
     */
    close() {
        return this.#browserFrame.page.close();
    }
    /**
     * Sets the URL without navigating the browser.
     *
     * @param url URL.
     */
    setURL(url) {
        this.#browserFrame.url = url;
    }
    /**
     * Sets the viewport.
     *
     * @param viewport Viewport.
     */
    setViewport(viewport) {
        this.#browserFrame.page.setViewport(viewport);
    }
    /**
     * Sets the window size.
     *
     * @deprecated Use setViewport() instead.
     * @param options Options.
     * @param options.width Width.
     * @param options.height Height.
     */
    setWindowSize(options) {
        this.setViewport({
            width: options?.width,
            height: options?.height
        });
    }
    /**
     * Sets the window width.
     *
     * @deprecated Use setViewport() instead.
     * @param width Width.
     */
    setInnerWidth(width) {
        this.setViewport({ width });
    }
    /**
     * Sets the window height.
     *
     * @deprecated Use setViewport() instead.
     * @param height Height.
     */
    setInnerHeight(height) {
        this.setViewport({ height });
    }
}
//# sourceMappingURL=DetachedWindowAPI.js.map