import IBrowserFrame from '../types/IBrowserFrame.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import IGoToOptions from '../types/IGoToOptions.js';
import Response from '../../fetch/Response.js';
import DocumentReadyStateManager from '../../nodes/document/DocumentReadyStateManager.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import BrowserFrameFactory from './BrowserFrameFactory.js';
import BrowserFrameURL from './BrowserFrameURL.js';
import BrowserFrameValidator from './BrowserFrameValidator.js';
import AsyncTaskManager from '../../async-task-manager/AsyncTaskManager.js';
import FormData from '../../form-data/FormData.js';
import HistoryScrollRestorationEnum from '../../history/HistoryScrollRestorationEnum.js';
import IHistoryItem from '../../history/IHistoryItem.js';

/**
 * Browser frame navigation utility.
 */
export default class BrowserFrameNavigator {
	/**
	 * Navigates to a page.
	 *
	 * @throws Error if the request can't be resolved (because of SSL error or similar). It will not throw if the response is not ok.
	 * @param options Options.
	 * @param options.windowClass Window class.
	 * @param options.frame Frame.
	 * @param options.url URL.
	 * @param [options.goToOptions] Go to options.
	 * @param [options.method] Method.
	 * @param [options.formData] Form data.
	 * @param [options.disableHistory] Disables adding the navigation to the history.
	 * @returns Response.
	 */
	public static async navigate(options: {
		windowClass: new (
			browserFrame: IBrowserFrame,
			options?: { url?: string; width?: number; height?: number }
		) => BrowserWindow | null;
		frame: IBrowserFrame;
		url: string;
		goToOptions?: IGoToOptions;
		method?: string;
		formData?: FormData | null;
		disableHistory?: boolean;
	}): Promise<Response | null> {
		const { windowClass, frame, url, formData, method, goToOptions, disableHistory } = options;
		const exceptionObserver = frame.page.context.browser[PropertySymbol.exceptionObserver];
		const referrer = goToOptions?.referrer || frame.window.location.origin;
		const targetURL = BrowserFrameURL.getRelativeURL(frame, url);
		const resolveNavigationListeners = (): void => {
			const listeners = frame[PropertySymbol.listeners].navigation;
			frame[PropertySymbol.listeners].navigation = [];
			for (const listener of listeners) {
				listener();
			}
		};

		if (!frame.window) {
			throw new Error('The frame has been destroyed, the "window" property is not set.');
		}

		// Javascript protocol
		if (targetURL.protocol === 'javascript:') {
			if (frame && !frame.page.context.browser.settings.disableJavaScriptEvaluation) {
				const readyStateManager = (<
					{ [PropertySymbol.readyStateManager]: DocumentReadyStateManager }
				>(<unknown>frame.window))[PropertySymbol.readyStateManager];

				readyStateManager.startTask();
				const code =
					'//# sourceURL=' + frame.url + '\n' + targetURL.href.replace('javascript:', '');

				// The browser will wait for the next tick before executing the script.
				// Fixes issue where evaluating the response can throw an error.
				// By using requestAnimationFrame() the error will not reject the promise.
				// The error will be caught by process error level listener or a try and catch in the requestAnimationFrame().
				await new Promise((resolve) => {
					frame.window.requestAnimationFrame(() => {
						frame.window.requestAnimationFrame(resolve);
						frame.window.eval(code);
					});
				});

				readyStateManager.endTask();
				resolveNavigationListeners();
			}

			return null;
		}

		// Validate navigation
		if (!BrowserFrameValidator.validateCrossOriginPolicy(frame, targetURL)) {
			return null;
		}

		if (!BrowserFrameValidator.validateFrameNavigation(frame)) {
			if (!frame.page.context.browser.settings.navigation.disableFallbackToSetURL) {
				frame.window.location[PropertySymbol.setURL](frame, targetURL.href);
			}

			return null;
		}

		// History management.
		if (!disableHistory) {
			const history = frame[PropertySymbol.history];

			for (let i = history.length - 1; i >= 0; i--) {
				if (history[i].isCurrent) {
					history[i].isCurrent = false;

					// We need to remove all history items after the current one.
					history.length = i + 1;
					break;
				}
			}

			history.push({
				title: '',
				href: targetURL.href,
				state: null,
				scrollRestoration: HistoryScrollRestorationEnum.auto,
				method: method || (formData ? 'POST' : 'GET'),
				formData: formData || null,
				isCurrent: true
			});
		}

		// Store current Window state
		const previousWindow = frame.window;
		const previousAsyncTaskManager = frame[PropertySymbol.asyncTaskManager];
		const width = previousWindow.innerWidth;
		const height = previousWindow.innerHeight;
		const devicePixelRatio = previousWindow.devicePixelRatio;
		const parentWindow = frame.parentFrame ? frame.parentFrame.window : frame.page.mainFrame.window;
		const topWindow = frame.page.mainFrame.window;

		// Create new Window
		frame[PropertySymbol.asyncTaskManager] = new AsyncTaskManager(frame);
		(<BrowserWindow>frame.window) = new windowClass(frame, { url: targetURL.href, width, height });
		frame.window[PropertySymbol.parent] = parentWindow;
		frame.window[PropertySymbol.top] = topWindow;
		(<number>frame.window.devicePixelRatio) = devicePixelRatio;

		if (exceptionObserver) {
			exceptionObserver.observe(frame.window);
		}

		if (referrer) {
			frame.window.document[PropertySymbol.referrer] = referrer;
		}

		// Destroy child frames and Window
		const destroyTaskID = frame[PropertySymbol.asyncTaskManager].startTask();
		const destroyWindowAndAsyncTaskManager = (): void => {
			previousAsyncTaskManager.destroy().then(() => {
				if (exceptionObserver) {
					exceptionObserver.disconnect(previousWindow);
				}
				frame[PropertySymbol.asyncTaskManager].endTask(destroyTaskID);
			});

			previousWindow[PropertySymbol.destroy]();
		};

		if (frame.childFrames.length) {
			Promise.all(
				frame.childFrames.map((childFrame) => BrowserFrameFactory.destroyFrame(childFrame))
			).then(destroyWindowAndAsyncTaskManager);
		} else {
			destroyWindowAndAsyncTaskManager();
		}

		// About protocol
		if (targetURL.protocol === 'about:') {
			await new Promise((resolve) => frame.page.mainFrame.window.requestAnimationFrame(resolve));
			resolveNavigationListeners();
			return null;
		}

		// Start navigation
		const readyStateManager = (<{ [PropertySymbol.readyStateManager]: DocumentReadyStateManager }>(
			(<unknown>frame.window)
		))[PropertySymbol.readyStateManager];
		const abortController = new frame.window.AbortController();
		const timeout = frame.window.setTimeout(
			() => abortController.abort(new Error('Request timed out.')),
			goToOptions?.timeout ?? 30000
		);
		const finalize = (): void => {
			frame.window.clearTimeout(timeout);
			readyStateManager.endTask();
			resolveNavigationListeners();
		};
		let response: Response;
		let responseText: string;

		readyStateManager.startTask();

		try {
			response = await frame.window.fetch(targetURL.href, {
				referrer,
				referrerPolicy: goToOptions?.referrerPolicy || 'origin',
				signal: abortController.signal,
				method: method || (formData ? 'POST' : 'GET'),
				headers: goToOptions?.hard ? { 'Cache-Control': 'no-cache' } : undefined,
				body: formData
			});

			// Handles the "X-Frame-Options" header for child frames.
			if (frame.parentFrame) {
				const originURL = frame.parentFrame.window.location;
				const xFrameOptions = response.headers?.get('X-Frame-Options')?.toLowerCase();
				const isSameOrigin = originURL.origin === targetURL.origin || targetURL.origin === 'null';

				if (xFrameOptions === 'deny' || (xFrameOptions === 'sameorigin' && !isSameOrigin)) {
					throw new Error(
						`Refused to display '${url}' in a frame because it set 'X-Frame-Options' to '${xFrameOptions}'.`
					);
				}
			}

			responseText = await response.text();
		} catch (error) {
			finalize();
			throw error;
		}

		if (response.url) {
			frame.window[PropertySymbol.location][PropertySymbol.setURL](frame, response.url);
		}

		if (!response.ok) {
			frame.page.console.error(`GET ${targetURL.href} ${response.status} (${response.statusText})`);
		}

		// The frame may be destroyed during teardown.
		if (!frame.window) {
			return null;
		}

		// Fixes issue where evaluating the response can throw an error.
		// By using requestAnimationFrame() the error will not reject the promise.
		// The error will be caught by process error level listener or a try and catch in the requestAnimationFrame().
		await new Promise((resolve) => {
			frame.window.requestAnimationFrame(() => {
				frame.window.requestAnimationFrame(resolve);
				frame.content = responseText;
			});
		});

		finalize();

		return response;
	}

	/**
	 * Navigates back in history.
	 *
	 * @param options Options.
	 * @param options.windowClass Window class.
	 * @param options.frame Frame.
	 * @param [options.goToOptions] Go to options.
	 */
	public static navigateBack(options: {
		windowClass: new (
			browserFrame: IBrowserFrame,
			options?: { url?: string; width?: number; height?: number }
		) => BrowserWindow | null;
		frame: IBrowserFrame;
		goToOptions?: IGoToOptions;
	}): Promise<Response | null> {
		const { windowClass, frame, goToOptions } = options;
		const history = frame[PropertySymbol.history];
		let historyItem: IHistoryItem;

		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].isCurrent) {
				if (i > 0) {
					history[i].isCurrent = false;
					historyItem = history[i - 1];
				}
				break;
			}
		}

		if (!historyItem) {
			return new Promise((resolve) => {
				frame.window.requestAnimationFrame(() => {
					const listeners = frame[PropertySymbol.listeners].navigation;
					frame[PropertySymbol.listeners].navigation = [];
					for (const listener of listeners) {
						listener();
					}
					resolve(null);
				});
			});
		}

		historyItem.isCurrent = true;

		return BrowserFrameNavigator.navigate({
			windowClass,
			frame,
			goToOptions: {
				...goToOptions,
				referrer: frame.url
			},
			url: historyItem.href,
			method: historyItem.method,
			formData: historyItem.formData,
			disableHistory: true
		});
	}

	/**
	 * Navigates forward in history.
	 *
	 * @param options Options.
	 * @param options.windowClass Window class.
	 * @param options.frame Frame.
	 * @param [options.goToOptions] Go to options.
	 */
	public static navigateForward(options: {
		windowClass: new (
			browserFrame: IBrowserFrame,
			options?: { url?: string; width?: number; height?: number }
		) => BrowserWindow | null;
		frame: IBrowserFrame;
		goToOptions?: IGoToOptions;
	}): Promise<Response | null> {
		const { windowClass, frame, goToOptions } = options;
		const history = frame[PropertySymbol.history];
		let historyItem: IHistoryItem;

		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].isCurrent) {
				if (i < history.length - 1) {
					history[i].isCurrent = false;
					historyItem = history[i + 1];
				}
				break;
			}
		}

		if (!historyItem) {
			return new Promise((resolve) => {
				frame.window.requestAnimationFrame(() => {
					const listeners = frame[PropertySymbol.listeners].navigation;
					frame[PropertySymbol.listeners].navigation = [];
					for (const listener of listeners) {
						listener();
					}
					resolve(null);
				});
			});
		}

		historyItem.isCurrent = true;

		return BrowserFrameNavigator.navigate({
			windowClass,
			frame,
			goToOptions: {
				...goToOptions,
				referrer: frame.url
			},
			url: historyItem.href,
			method: historyItem.method,
			formData: historyItem.formData,
			disableHistory: true
		});
	}

	/**
	 * Navigates steps in history.
	 *
	 * @param options Options.
	 * @param options.windowClass Window class.
	 * @param options.frame Frame.
	 * @param options.goToOptions Go to options.
	 * @param options.steps Steps.
	 */
	public static navigateSteps(options: {
		windowClass: new (
			browserFrame: IBrowserFrame,
			options?: { url?: string; width?: number; height?: number }
		) => BrowserWindow | null;
		frame: IBrowserFrame;
		goToOptions?: IGoToOptions;
		steps?: number;
	}): Promise<Response | null> {
		if (!options.steps) {
			return this.reload(options);
		}

		const { windowClass, frame, goToOptions, steps } = options;
		const history = frame[PropertySymbol.history];
		let historyItem: IHistoryItem;

		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].isCurrent) {
				if (history[i + steps]) {
					history[i].isCurrent = false;
					historyItem = history[i + steps];
				}
				break;
			}
		}

		if (!historyItem) {
			return new Promise((resolve) => {
				frame.window.requestAnimationFrame(() => {
					const listeners = frame[PropertySymbol.listeners].navigation;
					frame[PropertySymbol.listeners].navigation = [];
					for (const listener of listeners) {
						listener();
					}
					resolve(null);
				});
			});
		}

		historyItem.isCurrent = true;

		return BrowserFrameNavigator.navigate({
			windowClass,
			frame,
			goToOptions: {
				...goToOptions,
				referrer: frame.url
			},
			url: historyItem.href,
			method: historyItem.method,
			formData: historyItem.formData,
			disableHistory: true
		});
	}

	/**
	 * Reloads the current history item.
	 *
	 * @param options Options.
	 * @param options.windowClass Window class.
	 * @param options.frame Frame.
	 * @param options.goToOptions Go to options.
	 */
	public static reload(options: {
		windowClass: new (
			browserFrame: IBrowserFrame,
			options?: { url?: string; width?: number; height?: number }
		) => BrowserWindow | null;
		frame: IBrowserFrame;
		goToOptions?: IGoToOptions;
	}): Promise<Response | null> {
		const { windowClass, frame, goToOptions } = options;
		const history = frame[PropertySymbol.history];
		let historyItem: IHistoryItem;

		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].isCurrent) {
				historyItem = history[i];
				break;
			}
		}

		if (!historyItem) {
			return new Promise((resolve) => {
				frame.window.requestAnimationFrame(() => {
					const listeners = frame[PropertySymbol.listeners].navigation;
					frame[PropertySymbol.listeners].navigation = [];
					for (const listener of listeners) {
						listener();
					}
					resolve(null);
				});
			});
		}

		return BrowserFrameNavigator.navigate({
			windowClass,
			frame,
			goToOptions: {
				...goToOptions,
				referrer: frame.url
			},
			url: historyItem.href,
			method: historyItem.method,
			formData: historyItem.formData,
			disableHistory: true
		});
	}
}
