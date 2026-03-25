import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Event from '../../event/Event.js';
import ErrorEvent from '../../event/events/ErrorEvent.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import WindowBrowserContext from '../../window/WindowBrowserContext.js';
import BrowserErrorCaptureEnum from '../../browser/enums/BrowserErrorCaptureEnum.js';
import Attr from '../attr/Attr.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import ResourceFetch from '../../fetch/ResourceFetch.js';
import DocumentReadyStateManager from '../document/DocumentReadyStateManager.js';

/**
 * HTML Script Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLScriptElement.
 */
export default class HTMLScriptElement extends HTMLElement {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLScriptElement;

	// Events
	public onerror: (event: ErrorEvent) => void = null;
	public onload: (event: Event) => void = null;

	// Internal properties
	public [PropertySymbol.evaluateScript] = true;

	// Private properties
	#loadedScriptURL: string | null = null;
	/**
	 * Returns type.
	 *
	 * @returns Type.
	 */
	public get type(): string {
		return this.getAttribute('type') || '';
	}

	/**
	 * Sets type.
	 *
	 * @param type Type.
	 */
	public set type(type: string) {
		this.setAttribute('type', type);
	}

	/**
	 * Returns source.
	 *
	 * @returns Source.
	 */
	public get src(): string {
		if (!this.hasAttribute('src')) {
			return '';
		}

		try {
			return new URL(this.getAttribute('src'), this[PropertySymbol.ownerDocument].location.href)
				.href;
		} catch (e) {
			return this.getAttribute('src');
		}
	}

	/**
	 * Sets source.
	 *
	 * @param src Source.
	 */
	public set src(src: string) {
		this.setAttribute('src', src);
	}

	/**
	 * Returns charset.
	 *
	 * @returns Charset.
	 */
	public get charset(): string {
		return this.getAttribute('charset') || '';
	}

	/**
	 * Sets charset.
	 *
	 * @param charset Charset.
	 */
	public set charset(charset: string) {
		this.setAttribute('charset', charset);
	}

	/**
	 * Returns lang.
	 *
	 * @returns Lang.
	 */
	public get lang(): string {
		return this.getAttribute('lang') || '';
	}

	/**
	 * Sets lang.
	 *
	 * @param lang Lang.
	 */
	public set lang(lang: string) {
		this.setAttribute('lang', lang);
	}

	/**
	 * Returns async.
	 *
	 * @returns Async.
	 */
	public get async(): boolean {
		return this.getAttribute('async') !== null;
	}

	/**
	 * Sets async.
	 *
	 * @param async Async.
	 */
	public set async(async: boolean) {
		if (!async) {
			this.removeAttribute('async');
		} else {
			this.setAttribute('async', '');
		}
	}

	/**
	 * Returns defer.
	 *
	 * @returns Defer.
	 */
	public get defer(): boolean {
		return this.getAttribute('defer') !== null;
	}

	/**
	 * Sets defer.
	 *
	 * @param defer Defer.
	 */
	public set defer(defer: boolean) {
		if (!defer) {
			this.removeAttribute('defer');
		} else {
			this.setAttribute('defer', '');
		}
	}

	/**
	 * Returns text.
	 *
	 * @returns Text.
	 */
	public get text(): string {
		return this.textContent;
	}

	/**
	 * Sets text.
	 *
	 * @param text Text.
	 */
	public set text(text: string) {
		this.textContent = text;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLScriptElement {
		return <HTMLScriptElement>super[PropertySymbol.cloneNode](deep);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.connectedToDocument](): void {
		const browserSettings = new WindowBrowserContext(this[PropertySymbol.window]).getSettings();

		super[PropertySymbol.connectedToDocument]();

		if (this[PropertySymbol.evaluateScript]) {
			const src = this.getAttribute('src');

			if (src !== null) {
				this.#loadScript(src);
			} else if (browserSettings && !browserSettings.disableJavaScriptEvaluation) {
				const textContent = this.textContent;
				const type = this.getAttribute('type');
				if (
					textContent &&
					(type === null ||
						type === 'application/x-ecmascript' ||
						type === 'application/x-javascript' ||
						type.startsWith('text/javascript'))
				) {
					this[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = this;

					const code = `//# sourceURL=${this[PropertySymbol.window].location.href}\n` + textContent;

					if (
						browserSettings.disableErrorCapturing ||
						browserSettings.errorCapture !== BrowserErrorCaptureEnum.tryAndCatch
					) {
						this[PropertySymbol.window].eval(code);
					} else {
						WindowErrorUtility.captureError(this[PropertySymbol.window], () =>
							this[PropertySymbol.window].eval(code)
						);
					}

					this[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = null;
				}
			}
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onSetAttribute](
		attribute: Attr,
		replacedAttribute: Attr | null
	): void {
		super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);

		if (
			attribute[PropertySymbol.name] === 'src' &&
			attribute[PropertySymbol.value] !== null &&
			this[PropertySymbol.isConnected]
		) {
			this.#loadScript(attribute[PropertySymbol.value]);
		}
	}

	/**
	 * Returns a URL relative to the given Location object.
	 *
	 * @param url URL.
	 */
	async #loadScript(url: string): Promise<void> {
		const window = this[PropertySymbol.window];
		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();
		const async =
			this.getAttribute('async') !== null ||
			this.getAttribute('defer') !== null ||
			this.getAttribute('type') === 'module';

		if (!browserFrame) {
			return;
		}

		const browserSettings = browserFrame.page?.context?.browser?.settings;

		if (!url || !this[PropertySymbol.isConnected]) {
			return;
		}

		let absoluteURL: string;
		try {
			absoluteURL = new URL(url, this[PropertySymbol.window].location.href).href;
		} catch (error) {
			return;
		}

		if (this.#loadedScriptURL === absoluteURL) {
			return;
		}

		if (
			browserSettings &&
			(browserSettings.disableJavaScriptFileLoading || browserSettings.disableJavaScriptEvaluation)
		) {
			if (browserSettings.handleDisabledFileLoadingAsSuccess) {
				this.dispatchEvent(new Event('load'));
			} else {
				WindowErrorUtility.dispatchError(
					this,
					new window.DOMException(
						`Failed to load external script "${absoluteURL}". JavaScript file loading is disabled.`,
						DOMExceptionNameEnum.notSupportedError
					)
				);
			}
			return;
		}

		const resourceFetch = new ResourceFetch({
			browserFrame,
			window: this[PropertySymbol.window]
		});
		let code: string | null = null;
		let error: Error | null = null;

		this.#loadedScriptURL = absoluteURL;

		if (async) {
			const readyStateManager = (<
				{ [PropertySymbol.readyStateManager]: DocumentReadyStateManager }
			>(<unknown>this[PropertySymbol.window]))[PropertySymbol.readyStateManager];

			readyStateManager.startTask();

			try {
				code = await resourceFetch.fetch(absoluteURL);
			} catch (e) {
				error = e;
			}

			readyStateManager.endTask();
		} else {
			try {
				code = resourceFetch.fetchSync(absoluteURL);
			} catch (e) {
				error = e;
			}
		}

		if (error) {
			WindowErrorUtility.dispatchError(this, error);
		} else {
			this[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = this;
			code = '//# sourceURL=' + absoluteURL + '\n' + code;

			if (
				browserSettings.disableErrorCapturing ||
				browserSettings.errorCapture !== BrowserErrorCaptureEnum.tryAndCatch
			) {
				this[PropertySymbol.window].eval(code);
			} else {
				WindowErrorUtility.captureError(this[PropertySymbol.window], () =>
					this[PropertySymbol.window].eval(code)
				);
			}
			this[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = null;
			this.dispatchEvent(new Event('load'));
		}
	}
}
