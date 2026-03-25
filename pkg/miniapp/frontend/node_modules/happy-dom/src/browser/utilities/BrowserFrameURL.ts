import IBrowserFrame from '../types/IBrowserFrame.js';
import { URL } from 'url';

/**
 * Browser frame URL utility.
 */
export default class BrowserFrameURL {
	/**
	 * Returns relative URL.
	 *
	 * @param frame Frame.
	 * @param url URL.
	 * @returns Relative URL.
	 */
	public static getRelativeURL(frame: IBrowserFrame, url: string | URL): URL {
		url = url ? String(url) : 'about:blank';

		if (url.startsWith('about:') || url.startsWith('javascript:')) {
			return new URL(url);
		}

		try {
			return new URL(url, frame.window.location.href);
		} catch (e) {
			return new URL('about:blank');
		}
	}
}
