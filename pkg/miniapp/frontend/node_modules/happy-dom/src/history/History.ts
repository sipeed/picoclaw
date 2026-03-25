import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import HistoryScrollRestorationEnum from './HistoryScrollRestorationEnum.js';
import * as PropertySymbol from '../PropertySymbol.js';
import IHistoryItem from './IHistoryItem.js';
import BrowserFrameURL from '../browser/utilities/BrowserFrameURL.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * History API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/History.
 */
export default class History {
	#browserFrame: IBrowserFrame;
	#window: BrowserWindow;
	#currentHistoryItem: IHistoryItem;

	/**
	 * Constructor.
	 *
	 * @param browserFrame Browser frame.
	 * @param window Owner window.
	 */
	constructor(browserFrame: IBrowserFrame, window: BrowserWindow) {
		if (!browserFrame) {
			throw new TypeError('Illegal constructor');
		}

		this.#browserFrame = browserFrame;
		this.#window = window;

		const history = browserFrame[PropertySymbol.history];

		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].isCurrent) {
				this.#currentHistoryItem = history[i];
				break;
			}
		}
	}

	/**
	 * Returns the history length.
	 *
	 * @returns History length.
	 */
	public get length(): number {
		return this.#browserFrame?.[PropertySymbol.history].length || 0;
	}

	/**
	 * Returns an any value representing the state at the top of the history stack. This is a way to look at the state without having to wait for a popstate event.
	 *
	 * @returns State.
	 */
	public get state(): object | null {
		return this.#currentHistoryItem.state;
	}

	/**
	 * Returns scroll restoration.
	 *
	 * @returns Sroll restoration.
	 */
	public get scrollRestoration(): HistoryScrollRestorationEnum {
		return this.#currentHistoryItem.scrollRestoration;
	}

	/**
	 * Sets scroll restoration.
	 *
	 * @param scrollRestoration Sroll restoration.
	 */
	public set scrollRestoration(scrollRestoration: HistoryScrollRestorationEnum) {
		switch (scrollRestoration) {
			case HistoryScrollRestorationEnum.auto:
			case HistoryScrollRestorationEnum.manual:
				this.#currentHistoryItem.scrollRestoration = scrollRestoration;
				break;
		}
	}

	/**
	 * Goes to the previous page in session history.
	 */
	public back(): void {
		if (!this.#window.closed) {
			this.#browserFrame?.goBack();
		}
	}

	/**
	 * Goes to the next page in session history.
	 */
	public forward(): void {
		if (!this.#window.closed) {
			this.#browserFrame?.goForward();
		}
	}

	/**
	 * Load a specific page from the session history.
	 *
	 * @param delta Delta.
	 * @param _delta
	 */
	public go(delta: number): void {
		if (!this.#window.closed) {
			this.#browserFrame?.goSteps(delta);
		}
	}

	/**
	 * Pushes the given data onto the session history stack.
	 *
	 * @param state State.
	 * @param title Title.
	 * @param [url] URL.
	 */
	public pushState(state: object, title, url?: string | URL): void {
		if (this.#window.closed) {
			return;
		}

		const history = this.#browserFrame?.[PropertySymbol.history];

		if (!history) {
			return;
		}

		const location = this.#window[PropertySymbol.location];
		const newURL = url ? BrowserFrameURL.getRelativeURL(this.#browserFrame, url) : location;

		if (url && newURL.origin !== location.origin) {
			throw new this.#window.DOMException(
				`Failed to execute 'pushState' on 'History': A history state object with URL '${url.toString()}' cannot be created in a document with origin '${location.origin}' and URL '${location.href}'.`,
				DOMExceptionNameEnum.securityError
			);
		}

		let previousHistoryItem;

		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].isCurrent) {
				previousHistoryItem = history[i];
				previousHistoryItem.isCurrent = false;

				// We need to remove all history items after the current one.
				history.length = i + 1;
				break;
			}
		}

		const newHistoryItem: IHistoryItem = {
			title: title || this.#window.document.title,
			href: newURL.href,
			state: JSON.parse(JSON.stringify(state)),
			scrollRestoration: this.#currentHistoryItem.scrollRestoration,
			method: previousHistoryItem?.method || 'GET',
			formData: previousHistoryItem?.formData || null,
			isCurrent: true
		};

		history.push(newHistoryItem);

		location[PropertySymbol.setURL](this.#browserFrame, newHistoryItem.href);

		this.#currentHistoryItem = newHistoryItem;
	}

	/**
	 * This method modifies the current history entry, replacing it with a new state.
	 *
	 * @param state State.
	 * @param title Title.
	 * @param [url] URL.
	 */
	public replaceState(state: object, title, url?: string | URL): void {
		if (this.#window.closed) {
			return;
		}

		const history = this.#browserFrame?.[PropertySymbol.history];

		if (!history) {
			return;
		}

		const location = this.#window[PropertySymbol.location];
		const newURL = url ? BrowserFrameURL.getRelativeURL(this.#browserFrame, url) : location;

		if (url && newURL.origin !== location.origin) {
			throw new this.#window.DOMException(
				`Failed to execute 'pushState' on 'History': A history state object with URL '${url.toString()}' cannot be created in a document with origin '${location.origin}' and URL '${location.href}'.`,
				DOMExceptionNameEnum.securityError
			);
		}

		for (let i = history.length - 1; i >= 0; i--) {
			if (history[i].isCurrent) {
				const newHistoryItem = {
					title: title || this.#window.document.title,
					href: newURL.href,
					state: JSON.parse(JSON.stringify(state)),
					scrollRestoration: history[i].scrollRestoration,
					method: history[i].method,
					formData: history[i].formData,
					isCurrent: true
				};

				history[i] = newHistoryItem;
				this.#currentHistoryItem = newHistoryItem;

				break;
			}
		}

		if (url) {
			location[PropertySymbol.setURL](this.#browserFrame, this.#currentHistoryItem.href);
		}
	}

	/**
	 * Destroys the history.
	 *
	 * This will make sure that the History API can't access page data from the next history item.
	 */
	public [PropertySymbol.destroy](): void {
		this.#browserFrame = null;
	}
}
