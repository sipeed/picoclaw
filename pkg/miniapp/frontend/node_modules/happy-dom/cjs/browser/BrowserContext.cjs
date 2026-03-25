"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CookieContainer_js_1 = __importDefault(require("../cookie/CookieContainer.cjs"));
const ResponseCache_js_1 = __importDefault(require("../fetch/cache/response/ResponseCache.cjs"));
const BrowserPage_js_1 = __importDefault(require("./BrowserPage.cjs"));
const PreflightResponseCache_js_1 = __importDefault(require("../fetch/cache/preflight/PreflightResponseCache.cjs"));
/**
 * Browser context.
 */
class BrowserContext {
    pages = [];
    browser;
    cookieContainer = new CookieContainer_js_1.default();
    responseCache = new ResponseCache_js_1.default();
    preflightResponseCache = new PreflightResponseCache_js_1.default();
    /**
     * Constructor.
     *
     * @param browser
     */
    constructor(browser) {
        this.browser = browser;
    }
    /**
     * Aborts all ongoing operations and destroys the context.
     */
    async close() {
        if (!this.browser) {
            return;
        }
        await Promise.all(this.pages.slice().map((page) => page.close()));
        const browser = this.browser;
        const index = browser.contexts.indexOf(this);
        if (index !== -1) {
            browser.contexts.splice(index, 1);
        }
        this.pages = [];
        this.browser = null;
        this.cookieContainer = null;
        this.responseCache.clear();
        this.preflightResponseCache.clear();
        this.responseCache = null;
        this.preflightResponseCache = null;
        if (browser.contexts.length === 0) {
            browser.close();
        }
    }
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     *
     * @returns Promise.
     */
    async waitUntilComplete() {
        await Promise.all(this.pages.map((page) => page.waitUntilComplete()));
    }
    /**
     * Aborts all ongoing operations.
     */
    abort() {
        return new Promise((resolve, reject) => {
            if (!this.pages.length) {
                resolve();
                return;
            }
            Promise.all(this.pages.slice().map((page) => page.abort()))
                .then(() => resolve())
                .catch((error) => reject(error));
        });
    }
    /**
     * Creates a new page.
     *
     * @returns Page.
     */
    newPage() {
        const page = new BrowserPage_js_1.default(this);
        this.pages.push(page);
        return page;
    }
}
exports.default = BrowserContext;
//# sourceMappingURL=BrowserContext.cjs.map