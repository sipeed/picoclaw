import DetachedWindowAPI from './DetachedWindowAPI.js';
import BrowserWindow from './BrowserWindow.js';
import DetachedBrowser from '../browser/detached-browser/DetachedBrowser.js';
/**
 * Window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
export default class Window extends BrowserWindow {
    // Detached Window API.
    happyDOM;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.width] Window width. Defaults to "1024".
     * @param [options.height] Window height. Defaults to "768".
     * @param [options.innerWidth] Inner width. Deprecated. Defaults to "1024".
     * @param [options.innerHeight] Inner height. Deprecated. Defaults to "768".
     * @param [options.url] URL.
     * @param [options.console] Console.
     * @param [options.settings] Settings.
     */
    constructor(options) {
        const browser = new DetachedBrowser(BrowserWindow, {
            console: options?.console,
            settings: options?.settings
        });
        const browserPage = browser.defaultContext.pages[0];
        const browserFrame = browserPage.mainFrame;
        if (options && (options.width || options.height || options.innerWidth || options.innerHeight)) {
            Object.assign(browserPage.viewport, {
                width: options.width || options.innerWidth || browserPage.viewport.width,
                height: options.height || options.innerHeight || browserPage.viewport.height
            });
        }
        super(browserFrame, {
            url: options?.url
        });
        browserFrame.window = this;
        this.happyDOM = new DetachedWindowAPI(browserFrame);
    }
}
//# sourceMappingURL=Window.js.map