import VirtualConsolePrinter from '../console/VirtualConsolePrinter.js';
import BrowserFrame from './BrowserFrame.js';
import VirtualConsole from '../console/VirtualConsole.js';
import BrowserPageUtility from './utilities/BrowserPageUtility.js';
import DefaultBrowserPageViewport from './DefaultBrowserPageViewport.js';
import Event from '../event/Event.js';
/**
 * Browser page.
 */
export default class BrowserPage {
    virtualConsolePrinter = new VirtualConsolePrinter();
    mainFrame;
    context;
    console;
    viewport = Object.assign({}, DefaultBrowserPageViewport);
    /**
     * Constructor.
     *
     * @param context Browser context.
     */
    constructor(context) {
        this.context = context;
        this.console = context.browser.console ?? new VirtualConsole(this.virtualConsolePrinter);
        this.mainFrame = new BrowserFrame(this);
    }
    /**
     * Returns frames.
     */
    get frames() {
        return BrowserPageUtility.getFrames(this);
    }
    /**
     * Returns the viewport.
     */
    get content() {
        return this.mainFrame.content;
    }
    /**
     * Sets the content.
     *
     * @param content Content.
     */
    set content(content) {
        this.mainFrame.content = content;
    }
    /**
     * Returns the URL.
     *
     * @returns URL.
     */
    get url() {
        return this.mainFrame.url;
    }
    /**
     * Sets the content.
     *
     * @param url URL.
     */
    set url(url) {
        this.mainFrame.url = url;
    }
    /**
     * Aborts all ongoing operations and destroys the page.
     */
    close() {
        return BrowserPageUtility.closePage(this);
    }
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     */
    waitUntilComplete() {
        return this.mainFrame.waitUntilComplete();
    }
    /**
     * Returns a promise that is resolved when the page has navigated and the response HTML has been written to the document.
     */
    waitForNavigation() {
        return this.mainFrame.waitForNavigation();
    }
    /**
     * Aborts all ongoing operations.
     */
    abort() {
        return this.mainFrame.abort();
    }
    /**
     * Evaluates code or a VM Script in the page's context.
     *
     * @param script Script.
     * @returns Result.
     */
    evaluate(script) {
        return this.mainFrame.evaluate(script);
    }
    /**
     * Sets the viewport.
     *
     * @param viewport Viewport.
     */
    setViewport(viewport) {
        const previousViewport = Object.assign({}, this.viewport);
        Object.assign(this.viewport, viewport);
        if (previousViewport.width !== this.viewport.width ||
            previousViewport.height !== this.viewport.height ||
            previousViewport.devicePixelRatio !== this.viewport.devicePixelRatio) {
            this.mainFrame.window.dispatchEvent(new Event('resize'));
        }
    }
    /**
     * Go to a page.
     *
     * @param url URL.
     * @param [options] Options.
     * @returns Response.
     */
    goto(url, options) {
        return this.mainFrame.goto(url, options);
    }
    /**
     * Navigates back in history.
     *
     * @param [options] Options.
     */
    goBack(options) {
        return this.mainFrame.goBack(options);
    }
    /**
     * Navigates forward in history.
     *
     * @param [options] Options.
     */
    goForward(options) {
        return this.mainFrame.goForward(options);
    }
    /**
     * Navigates a delta in history.
     *
     * @param delta Delta.
     * @param steps
     * @param [options] Options.
     */
    goSteps(steps, options) {
        return this.mainFrame.goSteps(steps, options);
    }
    /**
     * Reloads the current page.
     *
     * @param [options] Options.
     * @returns Response.
     */
    reload(options) {
        return this.mainFrame.reload(options);
    }
}
//# sourceMappingURL=BrowserPage.js.map