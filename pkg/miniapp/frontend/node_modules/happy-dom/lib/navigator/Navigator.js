import MimeTypeArray from './MimeTypeArray.js';
import PluginArray from './PluginArray.js';
import Permissions from '../permissions/Permissions.js';
import Clipboard from '../clipboard/Clipboard.js';
import WindowBrowserContext from '../window/WindowBrowserContext.js';
/**
 * Browser Navigator API.
 *
 * Mocked information is taken from FireFox.
 *
 * Reference:
 * https://html.spec.whatwg.org/multipage/system-state.html#dom-navigator.
 */
export default class Navigator {
    #window;
    #clipboard;
    #permissions;
    /**
     * Constructor.
     *
     * @param window Owner window.
     */
    constructor(window) {
        if (!window) {
            throw new TypeError('Invalid constructor');
        }
        this.#window = window;
        this.#clipboard = new Clipboard(window);
        this.#permissions = new Permissions(window);
    }
    /**
     * False if setting a cookie will be ignored and true otherwise.
     */
    get cookieEnabled() {
        return true;
    }
    /**
     * TODO: Not implemented.
     */
    get credentials() {
        return null;
    }
    /**
     * TODO: Not implemented.
     */
    get geolocation() {
        return null;
    }
    /**
     * String representing the preferred language of the user, usually the language of the browser UI.
     */
    get language() {
        return 'en-US';
    }
    /**
     * Array of string representing the user's preferred languages.
     */
    get languages() {
        return ['en-US', 'en'];
    }
    /**
     * TODO: Not implemented.
     */
    get locks() {
        return null;
    }
    /**
     * Maximum number of simultaneous touch contact points are supported by the current device.
     */
    get maxTouchPoints() {
        return new WindowBrowserContext(this.#window).getSettings()?.navigator.maxTouchPoints || 0;
    }
    /**
     * Number of logical processors available to run threads on the user's computer.
     */
    get hardwareConcurrency() {
        return 8;
    }
    /**
     * Browser app code name.
     */
    get appCodeName() {
        return 'Mozilla';
    }
    /**
     * Browser app name.
     */
    get appName() {
        return 'Netscape';
    }
    /**
     * Browser app version.
     */
    get appVersion() {
        const userAgent = this.userAgent;
        const index = userAgent.indexOf('/');
        return index > -1 ? userAgent.substring(index + 1) : '';
    }
    /**
     * Browser platform.
     */
    get platform() {
        const userAgent = this.userAgent;
        const indexStart = userAgent.indexOf('(');
        const indexEnd = userAgent.indexOf(')');
        return indexStart > -1 && indexEnd > -1 ? userAgent.substring(indexStart + 1, indexEnd) : '';
    }
    /**
     * Browser product.
     */
    get product() {
        return 'Gecko';
    }
    /**
     * Browser product sub.
     */
    get productSub() {
        return '20100101';
    }
    /**
     * Browser vendor.
     */
    get vendor() {
        return '';
    }
    /**
     * Browser vendor sub.
     */
    get vendorSub() {
        return '';
    }
    /**
     * Browser user agent.
     *
     * "appCodeName/appVersion number (Platform; Security; OS-or-CPU; Localization; rv: revision-version-number) product/productSub Application-Name Application-Name-version".
     */
    get userAgent() {
        return new WindowBrowserContext(this.#window).getSettings()?.navigator.userAgent || '';
    }
    /**
     * Boolean value indicating whether the browser is working online.
     */
    get onLine() {
        return true;
    }
    /**
     * Returns a Permissions object that can be used to query and update permission status of APIs covered by the Permissions API.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/permissions
     * @returns Permissions.
     */
    get permissions() {
        return this.#permissions;
    }
    /**
     * Returns a Clipboard object providing access to the contents of the system clipboard.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/clipboard
     * @returns Clipboard.
     */
    get clipboard() {
        return this.#clipboard;
    }
    /**
     * Boolean Indicates whether the user agent is controlled by automation.
     */
    get webdriver() {
        return true;
    }
    /**
     * The user's Do Not Track setting, which indicates whether the user is requesting web sites and advertisers to not track them.
     *
     * The value of the property reflects that of the DNT HTTP header, i.e. Values of "1", "0", or "unspecified".
     */
    get doNotTrack() {
        return 'unspecified';
    }
    /**
     * Browser mime-types.
     */
    get mimeTypes() {
        return new MimeTypeArray([]);
    }
    /**
     * Browser plugins.
     */
    get plugins() {
        return new PluginArray([]);
    }
    /**
     * Sends an HTTP POST request containing a small amount of data to a web server.
     *
     * @param url URL.
     * @param data Data.
     * @returns "true" if the user agent successfully queued the data for transfer. Otherwise, it returns "false".
     */
    sendBeacon(url, data) {
        this.#window.fetch(url, {
            method: 'POST',
            body: data
        });
        return true;
    }
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString() {
        return '[object Navigator]';
    }
}
//# sourceMappingURL=Navigator.js.map