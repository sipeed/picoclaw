import BrowserWindow from './BrowserWindow.js';
import CrossOriginBrowserWindow from './CrossOriginBrowserWindow.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import FetchCORSUtility from '../fetch/utilities/FetchCORSUtility.js';
import BrowserFrameURL from '../browser/utilities/BrowserFrameURL.js';
import { URL } from 'url';
import * as PropertySymbol from '../PropertySymbol.js';

/**
 * Window page open handler.
 */
export default class WindowPageOpenUtility {
	/**
	 * Opens a page.
	 *
	 * @param browserFrame Browser frame.
	 * @param [options] Options.
	 * @param [options.url] URL.
	 * @param [options.target] Target.
	 * @param [options.features] Window features.
	 */
	public static openPage(
		browserFrame: IBrowserFrame,
		options?: {
			url?: string;
			target?: string;
			features?: string;
		}
	): BrowserWindow | CrossOriginBrowserWindow | null {
		const features = this.getWindowFeatures(options?.features || '');
		const target = options?.target !== undefined ? String(options.target) : null;
		const originURL = new URL(browserFrame.window.location.href);
		const targetURL = BrowserFrameURL.getRelativeURL(browserFrame, options.url);
		const oldWindow = browserFrame.window;
		let targetFrame: IBrowserFrame;

		if (browserFrame.window !== oldWindow) {
			return null;
		}

		switch (target) {
			case '_self':
				targetFrame = browserFrame;
				break;
			case '_top':
				targetFrame = browserFrame.page.mainFrame;
				break;
			case '_parent':
				targetFrame = browserFrame.parentFrame ?? browserFrame;
				break;
			case '_blank':
			default:
				const newPage = browserFrame.page.context.newPage();
				targetFrame = newPage.mainFrame;
				targetFrame[PropertySymbol.openerFrame] = browserFrame;
				break;
		}

		targetFrame
			.goto(targetURL.href, {
				referrer: features.noreferrer ? undefined : browserFrame.window.location.origin,
				referrerPolicy: features.noreferrer ? 'no-referrer' : undefined
			})
			.catch((error) => {
				if (targetFrame.page?.console) {
					targetFrame.page.console.error(error);
				} else {
					throw error;
				}
			});

		if (targetURL.protocol === 'javascript:') {
			return targetFrame.window;
		}

		// When using a detached Window instance directly and not via the Browser API we will not navigate and the window for the frame will not have changed.
		if (targetFrame === browserFrame && browserFrame.window === oldWindow) {
			return null;
		}

		if (features.popup && target !== '_self' && target !== '_top' && target !== '_parent') {
			targetFrame[PropertySymbol.popup] = true;

			if (features?.width || features?.height) {
				targetFrame.page.setViewport({
					width: features?.width,
					height: features?.height
				});
			}

			if (features?.left) {
				(<number>targetFrame.window.screenLeft) = features.left;
				(<number>targetFrame.window.screenX) = features.left;
			}

			if (features?.top) {
				(<number>targetFrame.window.screenTop) = features.top;
				(<number>targetFrame.window.screenY) = features.top;
			}
		}

		if (
			target &&
			target !== '_self' &&
			target !== '_top' &&
			target !== '_parent' &&
			target !== '_blank'
		) {
			(<string>targetFrame.window.name) = target;
		}

		const isCORS = FetchCORSUtility.isCORS(originURL, targetFrame.url);

		if (
			!features.noopener &&
			!features.noreferrer &&
			browserFrame.window &&
			targetFrame[PropertySymbol.openerFrame] &&
			targetFrame.window !== browserFrame.window
		) {
			targetFrame[PropertySymbol.openerWindow] = isCORS
				? new CrossOriginBrowserWindow(browserFrame.window)
				: browserFrame.window;
		}

		if (features.noopener || features.noreferrer) {
			return null;
		}

		if (isCORS) {
			return new CrossOriginBrowserWindow(targetFrame.window, browserFrame.window);
		}

		return targetFrame.window;
	}

	/**
	 * Returns window features.
	 *
	 * @param features Window features string.
	 * @returns Window features.
	 */
	private static getWindowFeatures(features: string): {
		popup: boolean;
		width: number;
		height: number;
		left: number;
		top: number;
		noopener: boolean;
		noreferrer: boolean;
	} {
		const parts = features.split(',');
		const result: {
			popup: boolean;
			width: number;
			height: number;
			left: number;
			top: number;
			noopener: boolean;
			noreferrer: boolean;
		} = {
			popup: false,
			width: 0,
			height: 0,
			left: 0,
			top: 0,
			noopener: false,
			noreferrer: false
		};

		for (const part of parts) {
			const [key, value] = part.split('=');
			switch (key) {
				case 'popup':
					result.popup = !value || value === 'yes' || value === '1' || value === 'true';
					break;
				case 'width':
				case 'innerWidth':
					result.width = parseInt(value, 10);
					break;
				case 'height':
				case 'innerHeight':
					result.height = parseInt(value, 10);
					break;
				case 'left':
				case 'screenX':
					result.left = parseInt(value, 10);
					break;
				case 'top':
				case 'screenY':
					result.top = parseInt(value, 10);
					break;
				case 'noopener':
					result.noopener = true;
					break;
				case 'noreferrer':
					result.noreferrer = true;
					break;
			}
		}

		return result;
	}
}
