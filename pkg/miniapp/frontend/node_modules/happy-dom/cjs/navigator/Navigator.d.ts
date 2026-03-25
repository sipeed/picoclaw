import MimeTypeArray from './MimeTypeArray.cjs';
import PluginArray from './PluginArray.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
import Permissions from '../permissions/Permissions.cjs';
import Clipboard from '../clipboard/Clipboard.cjs';
import Blob from '../file/Blob.cjs';
import FormData from '../form-data/FormData.cjs';
/**
 * Browser Navigator API.
 *
 * Mocked information is taken from FireFox.
 *
 * Reference:
 * https://html.spec.whatwg.org/multipage/system-state.html#dom-navigator.
 */
export default class Navigator {
    #private;
    /**
     * Constructor.
     *
     * @param window Owner window.
     */
    constructor(window: BrowserWindow);
    /**
     * False if setting a cookie will be ignored and true otherwise.
     */
    get cookieEnabled(): boolean;
    /**
     * TODO: Not implemented.
     */
    get credentials(): string;
    /**
     * TODO: Not implemented.
     */
    get geolocation(): string;
    /**
     * String representing the preferred language of the user, usually the language of the browser UI.
     */
    get language(): string;
    /**
     * Array of string representing the user's preferred languages.
     */
    get languages(): string[];
    /**
     * TODO: Not implemented.
     */
    get locks(): string;
    /**
     * Maximum number of simultaneous touch contact points are supported by the current device.
     */
    get maxTouchPoints(): number;
    /**
     * Number of logical processors available to run threads on the user's computer.
     */
    get hardwareConcurrency(): number;
    /**
     * Browser app code name.
     */
    get appCodeName(): string;
    /**
     * Browser app name.
     */
    get appName(): string;
    /**
     * Browser app version.
     */
    get appVersion(): string;
    /**
     * Browser platform.
     */
    get platform(): string;
    /**
     * Browser product.
     */
    get product(): string;
    /**
     * Browser product sub.
     */
    get productSub(): string;
    /**
     * Browser vendor.
     */
    get vendor(): string;
    /**
     * Browser vendor sub.
     */
    get vendorSub(): string;
    /**
     * Browser user agent.
     *
     * "appCodeName/appVersion number (Platform; Security; OS-or-CPU; Localization; rv: revision-version-number) product/productSub Application-Name Application-Name-version".
     */
    get userAgent(): string;
    /**
     * Boolean value indicating whether the browser is working online.
     */
    get onLine(): boolean;
    /**
     * Returns a Permissions object that can be used to query and update permission status of APIs covered by the Permissions API.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/permissions
     * @returns Permissions.
     */
    get permissions(): Permissions;
    /**
     * Returns a Clipboard object providing access to the contents of the system clipboard.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/clipboard
     * @returns Clipboard.
     */
    get clipboard(): Clipboard;
    /**
     * Boolean Indicates whether the user agent is controlled by automation.
     */
    get webdriver(): boolean;
    /**
     * The user's Do Not Track setting, which indicates whether the user is requesting web sites and advertisers to not track them.
     *
     * The value of the property reflects that of the DNT HTTP header, i.e. Values of "1", "0", or "unspecified".
     */
    get doNotTrack(): string;
    /**
     * Browser mime-types.
     */
    get mimeTypes(): MimeTypeArray;
    /**
     * Browser plugins.
     */
    get plugins(): PluginArray;
    /**
     * Sends an HTTP POST request containing a small amount of data to a web server.
     *
     * @param url URL.
     * @param data Data.
     * @returns "true" if the user agent successfully queued the data for transfer. Otherwise, it returns "false".
     */
    sendBeacon(url: string, data: string | Blob | ArrayBuffer | ArrayBufferView | FormData): boolean;
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString(): string;
}
//# sourceMappingURL=Navigator.d.ts.map