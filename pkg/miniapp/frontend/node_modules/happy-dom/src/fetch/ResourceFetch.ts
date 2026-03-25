import BrowserWindow from '../window/BrowserWindow.js';
import URL from '../url/URL.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import Fetch from './Fetch.js';
import SyncFetch from './SyncFetch.js';

/**
 * Helper class for performing fetch of resources.
 */
export default class ResourceFetch {
	private window: BrowserWindow;
	#browserFrame: IBrowserFrame;

	/**
	 * Constructor.
	 *
	 * @param options Options.
	 * @param options.browserFrame Browser frame.
	 * @param options.window Window.
	 */
	constructor(options: { browserFrame: IBrowserFrame; window: BrowserWindow }) {
		this.#browserFrame = options.browserFrame;
		this.window = options.window;
	}

	/**
	 * Returns resource data asynchronously.
	 *
	 * @param url URL.
	 * @returns Response.
	 */
	public async fetch(url: string): Promise<string> {
		const fetch = new Fetch({
			browserFrame: this.#browserFrame,
			window: this.window,
			url,
			disableSameOriginPolicy: true
		});
		const response = await fetch.send();

		if (!response.ok) {
			throw new this.window.DOMException(
				`Failed to perform request to "${new URL(url, this.window.location.href).href}". Status ${
					response.status
				} ${response.statusText}.`
			);
		}

		return await response.text();
	}

	/**
	 * Returns resource data synchronously.
	 *
	 * @param url URL.
	 * @returns Response.
	 */
	public fetchSync(url: string): string {
		const fetch = new SyncFetch({
			browserFrame: this.#browserFrame,
			window: this.window,
			url,
			disableSameOriginPolicy: true
		});

		const response = fetch.send();

		if (!response.ok) {
			throw new this.window.DOMException(
				`Failed to perform request to "${new URL(url, this.window.location.href).href}". Status ${
					response.status
				} ${response.statusText}.`
			);
		}

		return response.body.toString();
	}
}
