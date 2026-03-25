import URL from '../url/URL.js';
import Fetch from './Fetch.js';
import SyncFetch from './SyncFetch.js';
/**
 * Helper class for performing fetch of resources.
 */
export default class ResourceFetch {
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
        const fetch = new Fetch({
            browserFrame: this.#browserFrame,
            window: this.window,
            url,
            disableSameOriginPolicy: true
        });
        const response = await fetch.send();
        if (!response.ok) {
            throw new this.window.DOMException(`Failed to perform request to "${new URL(url, this.window.location.href).href}". Status ${response.status} ${response.statusText}.`);
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
        const fetch = new SyncFetch({
            browserFrame: this.#browserFrame,
            window: this.window,
            url,
            disableSameOriginPolicy: true
        });
        const response = fetch.send();
        if (!response.ok) {
            throw new this.window.DOMException(`Failed to perform request to "${new URL(url, this.window.location.href).href}". Status ${response.status} ${response.statusText}.`);
        }
        return response.body.toString();
    }
}
//# sourceMappingURL=ResourceFetch.js.map