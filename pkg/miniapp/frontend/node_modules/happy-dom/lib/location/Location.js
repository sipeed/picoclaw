import HashChangeEvent from '../event/events/HashChangeEvent.js';
import * as PropertySymbol from '../PropertySymbol.js';
import { URL } from 'url';
/**
 * Location.
 */
export default class Location {
    // Public properties
    [Symbol.toStringTag] = 'Location';
    // Private properties
    #browserFrame;
    #url;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     * @param url URL.
     */
    constructor(browserFrame, url) {
        if (!browserFrame) {
            throw new TypeError('Illegal constructor');
        }
        this.#browserFrame = browserFrame;
        this.#url = new URL(url);
    }
    /**
     * Returns hash.
     *
     * @returns Hash.
     */
    get hash() {
        return this.#url.hash;
    }
    /**
     * Sets hash.
     *
     * @param hash Value.
     */
    set hash(hash) {
        if (!this.#browserFrame) {
            return;
        }
        const oldURL = this.#url.href;
        this.#url.hash = hash;
        const newURL = this.#url.href;
        if (newURL !== oldURL) {
            this.#browserFrame.window?.dispatchEvent(new HashChangeEvent('hashchange', { oldURL, newURL }));
            this.#browserFrame.window?.document?.[PropertySymbol.clearCache]();
        }
    }
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host() {
        return this.#url.host;
    }
    /**
     * Sets host.
     *
     * @param host Value.
     */
    set host(host) {
        const url = new URL(this.#url.href);
        url.host = host;
        this.href = url.href;
    }
    /**
     * Returns hostname.
     *
     * @returns Hostname.
     */
    get hostname() {
        return this.#url.hostname;
    }
    /**
     * Sets hostname.
     *
     * @param hostname Value.
     */
    set hostname(hostname) {
        const url = new URL(this.#url.href);
        url.hostname = hostname;
        this.href = url.href;
    }
    /**
     * Override set href.
     */
    get href() {
        return this.#url.href;
    }
    /**
     * Override set href.
     */
    set href(url) {
        if (!this.#browserFrame) {
            return;
        }
        this.#browserFrame.goto(url).catch((error) => {
            if (this.#browserFrame.page?.console) {
                this.#browserFrame.page.console.error(error);
            }
            else {
                throw error;
            }
        });
    }
    /**
     * Returns origin.
     *
     * @returns Origin.
     */
    get origin() {
        return this.#url.origin;
    }
    /**
     * Returns pathname
     *
     * @returns Pathname.
     */
    get pathname() {
        return this.#url.pathname;
    }
    /**
     * Sets pathname.
     *
     * @param pathname Value.
     */
    set pathname(pathname) {
        const url = new URL(this.#url.href);
        url.pathname = pathname;
        this.href = url.href;
    }
    /**
     * Returns port.
     *
     * @returns Port.
     */
    get port() {
        return this.#url.port;
    }
    /**
     * Sets port.
     *
     * @param port Value.
     */
    set port(port) {
        const url = new URL(this.#url.href);
        url.port = port;
        this.href = url.href;
    }
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    get protocol() {
        return this.#url.protocol;
    }
    /**
     * Sets protocol.
     *
     * @param protocol Value.
     */
    set protocol(protocol) {
        const url = new URL(this.#url.href);
        url.protocol = protocol;
        this.href = url.href;
    }
    /**
     * Returns search.
     *
     * @returns Search.
     */
    get search() {
        return this.#url.search;
    }
    /**
     * Sets search.
     *
     * @param search Value.
     */
    set search(search) {
        const url = new URL(this.#url.href);
        url.search = search;
        this.href = url.href;
    }
    /**
     * Replaces the current resource with the one at the provided URL. The difference from the assign() method is that after using replace() the current page will not be saved in session History, meaning the user won't be able to use the back button to navigate to it.
     *
     * @param url URL.
     */
    replace(url) {
        this.href = String(url);
    }
    /**
     * Loads the resource at the URL provided in parameter.
     *
     * @param url URL.
     */
    assign(url) {
        this.href = String(url);
    }
    /**
     * Reloads the resource from the current URL.
     */
    reload() {
        if (!this.#browserFrame) {
            return;
        }
        this.#browserFrame.goto(this.href).catch((error) => {
            if (this.#browserFrame.page?.console) {
                this.#browserFrame.page.console.error(error);
            }
            else {
                throw error;
            }
        });
    }
    /**
     * Replaces the current URL state with the provided one without navigating to the new URL.
     *
     * @param browserFrame Browser frame that must match the current one as validation.
     * @param url URL.
     */
    [PropertySymbol.setURL](browserFrame, url) {
        if (!this.#browserFrame) {
            return;
        }
        if (this.#browserFrame !== browserFrame) {
            throw new Error('Failed to set URL. Browser frame mismatch.');
        }
        this.#url.href = url;
    }
    /**
     * Destroys the location.
     */
    [PropertySymbol.destroy]() {
        this.#browserFrame = null;
    }
    /**
     * Returns the URL as a string.
     *
     * @returns URL as a string.
     */
    toString() {
        return this.#url.toString();
    }
}
//# sourceMappingURL=Location.js.map