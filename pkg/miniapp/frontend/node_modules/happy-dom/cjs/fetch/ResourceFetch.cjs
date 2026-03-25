"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const URL_js_1 = __importDefault(require("../url/URL.cjs"));
const Fetch_js_1 = __importDefault(require("./Fetch.cjs"));
const SyncFetch_js_1 = __importDefault(require("./SyncFetch.cjs"));
/**
 * Helper class for performing fetch of resources.
 */
class ResourceFetch {
    window;
    #browserFrame;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.browserFrame Browser frame.
     * @param options.window Window.
     */
    constructor(options) {
        this.#browserFrame = options.browserFrame;
        this.window = options.window;
    }
    /**
     * Returns resource data asynchronously.
     *
     * @param url URL.
     * @returns Response.
     */
    async fetch(url) {
        const fetch = new Fetch_js_1.default({
            browserFrame: this.#browserFrame,
            window: this.window,
            url,
            disableSameOriginPolicy: true
        });
        const response = await fetch.send();
        if (!response.ok) {
            throw new this.window.DOMException(`Failed to perform request to "${new URL_js_1.default(url, this.window.location.href).href}". Status ${response.status} ${response.statusText}.`);
        }
        return await response.text();
    }
    /**
     * Returns resource data synchronously.
     *
     * @param url URL.
     * @returns Response.
     */
    fetchSync(url) {
        const fetch = new SyncFetch_js_1.default({
            browserFrame: this.#browserFrame,
            window: this.window,
            url,
            disableSameOriginPolicy: true
        });
        const response = fetch.send();
        if (!response.ok) {
            throw new this.window.DOMException(`Failed to perform request to "${new URL_js_1.default(url, this.window.location.href).href}". Status ${response.status} ${response.statusText}.`);
        }
        return response.body.toString();
    }
}
exports.default = ResourceFetch;
//# sourceMappingURL=ResourceFetch.cjs.map