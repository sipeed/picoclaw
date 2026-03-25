import * as PropertySymbol from '../PropertySymbol.js';
import AsyncTaskManager from '../async-task-manager/AsyncTaskManager.js';
import BrowserWindow from '../window/BrowserWindow.js';
import BrowserFrameURL from './utilities/BrowserFrameURL.js';
import BrowserFrameScriptEvaluator from './utilities/BrowserFrameScriptEvaluator.js';
import BrowserFrameNavigator from './utilities/BrowserFrameNavigator.js';
import HistoryScrollRestorationEnum from '../history/HistoryScrollRestorationEnum.js';
/**
 * Browser frame.
 */
export default class BrowserFrame {
    childFrames = [];
    parentFrame = null;
    page;
    window;
    [PropertySymbol.asyncTaskManager] = new AsyncTaskManager(this);
    [PropertySymbol.listeners] = { navigation: [] };
    [PropertySymbol.openerFrame] = null;
    [PropertySymbol.openerWindow] = null;
    [PropertySymbol.popup] = false;
    [PropertySymbol.history] = [
        {
            title: '',
            href: 'about:blank',
            state: null,
            scrollRestoration: HistoryScrollRestorationEnum.auto,
            method: 'GET',
            formData: null,
            isCurrent: true
        }
    ];
    /**
     * Constructor.
     *
     * @param page Page.
     */
    constructor(page) {
        this.page = page;
        this.window = new BrowserWindow(this);
        // Attach process level error capturing.
        if (page.context.browser[PropertySymbol.exceptionObserver]) {
            page.context.browser[PropertySymbol.exceptionObserver].observe(this.window);
        }
    }
    /**
     * Returns the content.
     *
     * @returns Content.
     */
    get content() {
        return this.window.document.documentElement.outerHTML;
    }
    /**
     * Sets the content.
     *
     * @param content Content.
     */
    set content(content) {
        this.window.document[PropertySymbol.isFirstWrite] = true;
        this.window.document[PropertySymbol.isFirstWriteAfterOpen] = false;
        this.window.document.open();
        this.window.document.write(content);
    }
    /**
     * Returns the URL.
     *
     * @returns URL.
     */
    get url() {
        return this.window.location.href;
    }
    /**
     * Sets the content.
     *
     * @param url URL.
     */
    set url(url) {
        this.window[PropertySymbol.location][PropertySymbol.setURL](this, BrowserFrameURL.getRelativeURL(this, url).href);
    }
    /**
     * Returns document.
     *
     * @returns Document.
     */
    get document() {
        return this.window?.document ?? null;
    }
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     */
    async waitUntilComplete() {
        await Promise.all([
            this[PropertySymbol.asyncTaskManager].waitUntilComplete(),
            ...this.childFrames.map((frame) => frame.waitUntilComplete())
        ]);
    }
    /**
     * Returns a promise that is resolved when the frame has navigated and the response HTML has been written to the document.
     */
    waitForNavigation() {
        return new Promise((resolve) => this[PropertySymbol.listeners].navigation.push(resolve));
    }
    /**
     * Aborts all ongoing operations.
     */
    abort() {
        if (!this.childFrames.length) {
            return this[PropertySymbol.asyncTaskManager].abort();
        }
        return new Promise((resolve, reject) => {
            // Using Promise instead of async/await to prevent microtask
            Promise.all(this.childFrames
                .map((frame) => frame.abort())
                .concat([this[PropertySymbol.asyncTaskManager].abort()]))
                .then(() => resolve())
                .catch(reject);
        });
    }
    /**
     * Evaluates code or a VM Script in the page's context.
     *
     * @param script Script.
     * @returns Result.
     */
    evaluate(script) {
        return BrowserFrameScriptEvaluator.evaluate(this, script);
    }
    /**
     * Go to a page.
     *
     * @param url URL.
     * @param [options] Options.
     * @returns Response.
     */
    goto(url, options) {
        return BrowserFrameNavigator.navigate({
            windowClass: BrowserWindow,
            frame: this,
            url: url,
            goToOptions: options
        });
    }
    /**
     * Navigates back in history.
     *
     * @param [options] Options.
     */
    goBack(options) {
        return BrowserFrameNavigator.navigateBack({
            windowClass: BrowserWindow,
            frame: this,
            goToOptions: options
        });
    }
    /**
     * Navigates forward in history.
     *
     * @param [options] Options.
     */
    goForward(options) {
        return BrowserFrameNavigator.navigateForward({
            windowClass: BrowserWindow,
            frame: this,
            goToOptions: options
        });
    }
    /**
     * Navigates a delta in history.
     *
     * @param steps Steps.
     * @param [options] Options.
     */
    goSteps(steps, options) {
        return BrowserFrameNavigator.navigateSteps({
            windowClass: BrowserWindow,
            frame: this,
            steps: steps,
            goToOptions: options
        });
    }
    /**
     * Reloads the current frame.
     *
     * @param [options] Options.
     * @returns Response.
     */
    reload(options) {
        return BrowserFrameNavigator.reload({
            windowClass: BrowserWindow,
            frame: this,
            goToOptions: options
        });
    }
}
//# sourceMappingURL=BrowserFrame.js.map