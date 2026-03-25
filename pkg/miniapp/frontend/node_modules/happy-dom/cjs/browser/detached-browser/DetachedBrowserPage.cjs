"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const VirtualConsolePrinter_js_1 = __importDefault(require("../../console/VirtualConsolePrinter.cjs"));
const DetachedBrowserFrame_js_1 = __importDefault(require("./DetachedBrowserFrame.cjs"));
const VirtualConsole_js_1 = __importDefault(require("../../console/VirtualConsole.cjs"));
const BrowserPageUtility_js_1 = __importDefault(require("../utilities/BrowserPageUtility.cjs"));
const DefaultBrowserPageViewport_js_1 = __importDefault(require("../DefaultBrowserPageViewport.cjs"));
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
/**
 * Detached browser page used when constructing a Window instance without a browser.
 */
class DetachedBrowserPage {
    virtualConsolePrinter = new VirtualConsolePrinter_js_1.default();
    mainFrame;
    context;
    console;
    viewport = Object.assign({}, DefaultBrowserPageViewport_js_1.default);
    /**
     * Constructor.
     *
     * @param context Browser context.
     */
    constructor(context) {
        this.context = context;
        this.console = context.browser.console ?? new VirtualConsole_js_1.default(this.virtualConsolePrinter);
        this.mainFrame = new DetachedBrowserFrame_js_1.default(this);
    }
    /**
     * Returns frames.
     */
    get frames() {
        return BrowserPageUtility_js_1.default.getFrames(this);
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
        // Using Promise instead of async/await to prevent microtask
        return new Promise((resolve, reject) => {
            const context = this.context;
            BrowserPageUtility_js_1.default.closePage(this)
                .then(() => {
                // As we are in a detached page, a context or browser should not exist without a page as there are no references to them.
                if (context.pages[0] === this) {
                    context.close().then(resolve).catch(reject);
                }
                else {
                    resolve();
                }
            })
                .catch(reject);
        });
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
            this.mainFrame.window.dispatchEvent(new Event_js_1.default('resize'));
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
exports.default = DetachedBrowserPage;
//# sourceMappingURL=DetachedBrowserPage.cjs.map