import MimeTypeArray from './MimeTypeArray.js';
import PluginArray from './PluginArray.js';
import BrowserWindow from '../window/BrowserWindow.js';
import Permissions from '../permissions/Permissions.js';
import Clipboard from '../clipboard/Clipboard.js';
import WindowBrowserContext from '../window/WindowBrowserContext.js';
import Blob from '../file/Blob.js';
import FormData from '../form-data/FormData.js';

/**
 * Browser Navigator API.
 *
 * Mocked information is taken from FireFox.
 *
 * Reference:
 * https://html.spec.whatwg.org/multipage/system-state.html#dom-navigator.
 */
export default class Navigator {
	#window: BrowserWindow;
	#clipboard: Clipboard;
	#permissions: Permissions;

	/**
	 * Constructor.
	 *
	 * @param window Owner window.
	 */
	constructor(window: BrowserWindow) {
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
	public get cookieEnabled(): boolean {
		return true;
	}

	/**
	 * TODO: Not implemented.
	 */
	public get credentials(): string {
		return null;
	}

	/**
	 * TODO: Not implemented.
	 */
	public get geolocation(): string {
		return null;
	}

	/**
	 * String representing the preferred language of the user, usually the language of the browser UI.
	 */
	public get language(): string {
		return 'en-US';
	}

	/**
	 * Array of string representing the user's preferred languages.
	 */
	public get languages(): string[] {
		return ['en-US', 'en'];
	}

	/**
	 * TODO: Not implemented.
	 */
	public get locks(): string {
		return null;
	}

	/**
	 * Maximum number of simultaneous touch contact points are supported by the current device.
	 */
	public get maxTouchPoints(): number {
		return new WindowBrowserContext(this.#window).getSettings()?.navigator.maxTouchPoints || 0;
	}

	/**
	 * Number of logical processors available to run threads on the user's computer.
	 */
	public get hardwareConcurrency(): number {
		return 8;
	}

	/**
	 * Browser app code name.
	 */
	public get appCodeName(): string {
		return 'Mozilla';
	}

	/**
	 * Browser app name.
	 */
	public get appName(): string {
		return 'Netscape';
	}

	/**
	 * Browser app version.
	 */
	public get appVersion(): string {
		const userAgent = this.userAgent;
		const index = userAgent.indexOf('/');
		return index > -1 ? userAgent.substring(index + 1) : '';
	}

	/**
	 * Browser platform.
	 */
	public get platform(): string {
		const userAgent = this.userAgent;
		const indexStart = userAgent.indexOf('(');
		const indexEnd = userAgent.indexOf(')');
		return indexStart > -1 && indexEnd > -1 ? userAgent.substring(indexStart + 1, indexEnd) : '';
	}

	/**
	 * Browser product.
	 */
	public get product(): string {
		return 'Gecko';
	}

	/**
	 * Browser product sub.
	 */
	public get productSub(): string {
		return '20100101';
	}

	/**
	 * Browser vendor.
	 */
	public get vendor(): string {
		return '';
	}

	/**
	 * Browser vendor sub.
	 */
	public get vendorSub(): string {
		return '';
	}

	/**
	 * Browser user agent.
	 *
	 * "appCodeName/appVersion number (Platform; Security; OS-or-CPU; Localization; rv: revision-version-number) product/productSub Application-Name Application-Name-version".
	 */
	public get userAgent(): string {
		return new WindowBrowserContext(this.#window).getSettings()?.navigator.userAgent || '';
	}

	/**
	 * Boolean value indicating whether the browser is working online.
	 */
	public get onLine(): boolean {
		return true;
	}

	/**
	 * Returns a Permissions object that can be used to query and update permission status of APIs covered by the Permissions API.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/permissions
	 * @returns Permissions.
	 */
	public get permissions(): Permissions {
		return this.#permissions;
	}

	/**
	 * Returns a Clipboard object providing access to the contents of the system clipboard.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/clipboard
	 * @returns Clipboard.
	 */
	public get clipboard(): Clipboard {
		return this.#clipboard;
	}

	/**
	 * Boolean Indicates whether the user agent is controlled by automation.
	 */
	public get webdriver(): boolean {
		return true;
	}

	/**
	 * The user's Do Not Track setting, which indicates whether the user is requesting web sites and advertisers to not track them.
	 *
	 * The value of the property reflects that of the DNT HTTP header, i.e. Values of "1", "0", or "unspecified".
	 */
	public get doNotTrack(): string {
		return 'unspecified';
	}

	/**
	 * Browser mime-types.
	 */
	public get mimeTypes(): MimeTypeArray {
		return new MimeTypeArray([]);
	}

	/**
	 * Browser plugins.
	 */
	public get plugins(): PluginArray {
		return new PluginArray([]);
	}

	/**
	 * Sends an HTTP POST request containing a small amount of data to a web server.
	 *
	 * @param url URL.
	 * @param data Data.
	 * @returns "true" if the user agent successfully queued the data for transfer. Otherwise, it returns "false".
	 */
	public sendBeacon(
		url: string,
		data: string | Blob | ArrayBuffer | ArrayBufferView | FormData
	): boolean {
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
	public toString(): string {
		return '[object Navigator]';
	}
}
