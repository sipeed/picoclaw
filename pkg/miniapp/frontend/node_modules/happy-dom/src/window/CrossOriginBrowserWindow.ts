import EventTarget from '../event/EventTarget.js';
import BrowserWindow from './BrowserWindow.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import Location from '../location/Location.js';
import * as PropertySymbol from '../PropertySymbol.js';

/**
 * Browser window with limited access due to CORS restrictions in iframes.
 */
export default class CrossOriginBrowserWindow
	extends EventTarget
	implements CrossOriginBrowserWindow
{
	public readonly window = this;
	public readonly location: Location;

	// Internal properties
	public [PropertySymbol.self]: BrowserWindow | CrossOriginBrowserWindow = this;
	public declare [PropertySymbol.top]: BrowserWindow | CrossOriginBrowserWindow;
	public declare [PropertySymbol.parent]: BrowserWindow | CrossOriginBrowserWindow;

	// Private properties
	#targetWindow: BrowserWindow;

	/**
	 * Constructor.
	 *
	 * @param target Target window.
	 * @param [parent] Parent window.
	 */
	constructor(target: BrowserWindow, parent?: BrowserWindow) {
		super();

		this[PropertySymbol.parent] = parent ?? this;
		this[PropertySymbol.top] = parent ?? this;

		this.location = <Location>new Proxy(
			{},
			{
				get: () => {
					throw new DOMException(
						`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`,
						DOMExceptionNameEnum.securityError
					);
				},
				set: () => {
					throw new DOMException(
						`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`,
						DOMExceptionNameEnum.securityError
					);
				}
			}
		);
		this.#targetWindow = target;
	}

	/**
	 * Returns self.
	 *
	 * @returns Self.
	 */
	public get self(): BrowserWindow | CrossOriginBrowserWindow {
		return this[PropertySymbol.self];
	}

	/**
	 * Returns self.
	 *
	 * @param self Self.
	 */
	public set self(self: BrowserWindow | CrossOriginBrowserWindow | null) {
		this[PropertySymbol.self] = self;
	}

	/**
	 * Returns top.
	 *
	 * @returns Top.
	 */
	public get top(): BrowserWindow | CrossOriginBrowserWindow {
		return this[PropertySymbol.top];
	}

	/**
	 * Returns parent.
	 *
	 * @returns Parent.
	 */
	public get parent(): BrowserWindow | CrossOriginBrowserWindow {
		return this[PropertySymbol.parent];
	}

	/**
	 * Returns parent.
	 *
	 * @param parent Parent.
	 */
	public set parent(parent: BrowserWindow | null) {
		this[PropertySymbol.parent] = parent;
	}

	/**
	 * Returns the opener.
	 *
	 * @returns Opener.
	 */
	public get opener(): BrowserWindow | CrossOriginBrowserWindow | null {
		return this.#targetWindow.opener;
	}

	/**
	 * Returns the closed state.
	 *
	 * @returns Closed state.
	 */
	public get closed(): boolean {
		return this.#targetWindow.closed;
	}

	/**
	 * Shifts focus away from the window.
	 */
	public blur(): void {
		this.#targetWindow.blur();
	}

	/**
	 * Gives focus to the window.
	 */
	public focus(): void {
		this.#targetWindow.focus();
	}

	/**
	 * Closes the window.
	 */
	public close(): void {
		this.#targetWindow.close();
	}

	/**
	 * Safely enables cross-origin communication between Window objects; e.g., between a page and a pop-up that it spawned, or between a page and an iframe embedded within it.
	 *
	 * @param message Message.
	 * @param [targetOrigin=*] Target origin.
	 * @param transfer Transfer. Not implemented.
	 */
	public postMessage(message: unknown, targetOrigin = '*', transfer?: unknown[]): void {
		this.#targetWindow.postMessage(message, targetOrigin, transfer);
	}
}
