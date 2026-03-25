"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DetachedWindowAPI_js_1 = __importDefault(require("./DetachedWindowAPI.cjs"));
const BrowserWindow_js_1 = __importDefault(require("./BrowserWindow.cjs"));
const DetachedBrowser_js_1 = __importDefault(require("../browser/detached-browser/DetachedBrowser.cjs"));
/**
 * Window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
class Window extends BrowserWindow_js_1.default {
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
        const browser = new DetachedBrowser_js_1.default(BrowserWindow_js_1.default, {
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
        this.happyDOM = new DetachedWindowAPI_js_1.default(browserFrame);
    }
}
exports.default = Window;
//# sourceMappingURL=Window.cjs.map