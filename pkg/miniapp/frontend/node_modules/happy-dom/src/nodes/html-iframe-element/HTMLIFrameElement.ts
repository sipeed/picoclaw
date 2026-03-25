import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import Document from '../document/Document.js';
import HTMLElement from '../html-element/HTMLElement.js';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.js';
import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import Attr from '../attr/Attr.js';
import BrowserFrameFactory from '../../browser/utilities/BrowserFrameFactory.js';
import BrowserFrameURL from '../../browser/utilities/BrowserFrameURL.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import IRequestReferrerPolicy from '../../fetch/types/IRequestReferrerPolicy.js';
import WindowBrowserContext from '../../window/WindowBrowserContext.js';

const SANDBOX_FLAGS = [
	'allow-downloads',
	'allow-forms',
	'allow-modals',
	'allow-orientation-lock',
	'allow-pointer-lock',
	'allow-popups',
	'allow-popups-to-escape-sandbox',
	'allow-presentation',
	'allow-same-origin',
	'allow-scripts',
	'allow-top-navigation',
	'allow-top-navigation-by-user-activation',
	'allow-top-navigation-to-custom-protocols'
];

/**
 * HTML Iframe Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLIFrameElement.
 */
export default class HTMLIFrameElement extends HTMLElement {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLIFrameElement;

	// Events
	public onload: ((event: Event) => void) | null = null;
	public onerror: ((event: Event) => void) | null = null;

	// Internal properties
	public [PropertySymbol.sandbox]: DOMTokenList | null = null;

	// Private properties
	#contentWindowContainer: { window: BrowserWindow | CrossOriginBrowserWindow | null } = {
		window: null
	};
	#iframe: IBrowserFrame;
	#loadedSrcdoc: string | null = null;

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
	 * Returns allow.
	 *
	 * @returns Allow.
	 */
	public get allow(): string {
		return this.getAttribute('allow') || '';
	}

	/**
	 * Sets allow.
	 *
	 * @param allow Allow.
	 */
	public set allow(allow: string) {
		this.setAttribute('allow', allow);
	}

	/**
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): string {
		return this.getAttribute('height') || '';
	}

	/**
	 * Sets height.
	 *
	 * @param height Height.
	 */
	public set height(height: string) {
		this.setAttribute('height', height);
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): string {
		return this.getAttribute('width') || '';
	}

	/**
	 * Sets width.
	 *
	 * @param width Width.
	 */
	public set width(width: string) {
		this.setAttribute('width', width);
	}

	/**
	 * Returns name.
	 *
	 * @returns Name.
	 */
	public get name(): string {
		return this.getAttribute('name') || '';
	}

	/**
	 * Sets name.
	 *
	 * @param name Name.
	 */
	public set name(name: string) {
		this.setAttribute('name', name);
	}

	/**
	 * Returns sandbox.
	 *
	 * @returns Sandbox.
	 */
	public get sandbox(): DOMTokenList {
		if (!this[PropertySymbol.sandbox]) {
			this[PropertySymbol.sandbox] = new DOMTokenList(
				PropertySymbol.illegalConstructor,
				this,
				'sandbox'
			);
		}
		return <DOMTokenList>this[PropertySymbol.sandbox];
	}

	/**
	 * Sets sandbox.
	 */
	public set sandbox(sandbox: string) {
		this.setAttribute('sandbox', sandbox);
	}

	/**
	 * Returns srcdoc.
	 *
	 * @returns Srcdoc.
	 */
	public get srcdoc(): string {
		return this.getAttribute('srcdoc') || '';
	}

	/**
	 * Sets srcdoc.
	 *
	 * @param srcdoc Srcdoc.
	 */
	public set srcdoc(srcdoc: string) {
		this.setAttribute('srcdoc', srcdoc);
	}

	/**
	 * Returns referrer policy.
	 */
	public get referrerPolicy(): string {
		return this.getAttribute('referrerpolicy') || '';
	}

	/**
	 * Sets referrer policy.
	 *
	 * @param referrerPolicy Referrer policy.
	 */
	public set referrerPolicy(referrerPolicy: string) {
		this.setAttribute('referrerpolicy', referrerPolicy);
	}

	/**
	 * Returns content document.
	 *
	 * @returns Content document.
	 */
	public get contentDocument(): Document | null {
		return (<BrowserWindow>this.#contentWindowContainer.window)?.document ?? null;
	}

	/**
	 * Returns content window.
	 *
	 * @returns Content window.
	 */
	public get contentWindow(): BrowserWindow | CrossOriginBrowserWindow | null {
		return this.#contentWindowContainer.window;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.connectedToDocument](): void {
		super[PropertySymbol.connectedToDocument]();
		this.#loadPage();
	}

	/**
	 * Called when disconnected from document.
	 * @param e
	 */
	public [PropertySymbol.disconnectedFromDocument](): void {
		super[PropertySymbol.disconnectedFromDocument]();
		this.#unloadPage();
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLIFrameElement {
		return <HTMLIFrameElement>super[PropertySymbol.cloneNode](deep);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onSetAttribute](
		attribute: Attr,
		replacedAttribute: Attr | null
	): void {
		super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
		if (attribute[PropertySymbol.name] === 'srcdoc') {
			this.#loadPage();
		}

		if (
			attribute[PropertySymbol.name] === 'src' &&
			attribute[PropertySymbol.value] &&
			!this.hasAttribute('srcdoc') &&
			attribute[PropertySymbol.value] !== replacedAttribute?.[PropertySymbol.value]
		) {
			this.#loadPage();
		}

		if (attribute[PropertySymbol.name] === 'sandbox') {
			this.#validateSandboxFlags();
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onRemoveAttribute](removedAttribute: Attr): void {
		super[PropertySymbol.onRemoveAttribute](removedAttribute);

		if (
			removedAttribute[PropertySymbol.name] === 'srcdoc' ||
			removedAttribute[PropertySymbol.name] === 'src'
		) {
			this.#loadPage();
		}
	}

	/**
	 *
	 * @param tokens
	 * @param vconsole
	 */
	#validateSandboxFlags(): void {
		const window = this[PropertySymbol.window];
		const invalidFlags: string[] = [];

		for (const token of this.sandbox) {
			if (!SANDBOX_FLAGS.includes(token)) {
				invalidFlags.push(token);
			}
		}

		if (invalidFlags.length === 1) {
			window.console.error(
				`Error while parsing the 'sandbox' attribute: '${invalidFlags[0]}' is an invalid sandbox flag.`
			);
		} else if (invalidFlags.length > 1) {
			window.console.error(
				`Error while parsing the 'sandbox' attribute: '${invalidFlags.join(
					`', '`
				)}' are invalid sandbox flags.`
			);
		}
	}

	/**
	 * Loads an iframe page.
	 */
	#loadPage(): void {
		if (!this[PropertySymbol.isConnected]) {
			this.#unloadPage();
			return;
		}

		const srcdoc = this.getAttribute('srcdoc');
		const window = this[PropertySymbol.window];
		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();

		if (!browserFrame) {
			return;
		}

		if (srcdoc !== null) {
			if (this.#loadedSrcdoc === srcdoc) {
				return;
			}

			this.#unloadPage();

			this.#iframe = BrowserFrameFactory.createChildFrame(browserFrame);
			this.#iframe.url = 'about:srcdoc';

			this.#contentWindowContainer.window = this.#iframe.window;

			this.#iframe.window[PropertySymbol.top] = browserFrame.window.top;
			this.#iframe.window[PropertySymbol.parent] = browserFrame.window;

			this.#iframe.window.document.open();
			this.#iframe.window.document.write(srcdoc);

			this.#loadedSrcdoc = srcdoc;

			this[PropertySymbol.window].requestAnimationFrame(() =>
				this.dispatchEvent(new Event('load'))
			);
			return;
		}

		if (this.#loadedSrcdoc !== null) {
			this.#unloadPage();
		}

		const originURL = browserFrame.window.location;
		const targetURL = BrowserFrameURL.getRelativeURL(browserFrame, this.src);

		if (this.#iframe && this.#iframe.window.location.href === targetURL.href) {
			return;
		}

		if (browserFrame.page.context.browser.settings.disableIframePageLoading) {
			WindowErrorUtility.dispatchError(
				this,
				new window.DOMException(
					`Failed to load iframe page "${targetURL.href}". Iframe page loading is disabled.`,
					DOMExceptionNameEnum.notSupportedError
				)
			);
			return;
		}

		// Iframes has a special rule for CORS and doesn't allow access between frames when the origin is different.
		const isSameOrigin = originURL.origin === targetURL.origin || targetURL.origin === 'null';
		const parentWindow = isSameOrigin ? window : new CrossOriginBrowserWindow(window);

		this.#iframe = this.#iframe ?? BrowserFrameFactory.createChildFrame(browserFrame);

		this.#iframe.window[PropertySymbol.top] = <BrowserWindow>parentWindow;
		this.#iframe.window[PropertySymbol.parent] = <BrowserWindow>parentWindow;

		this.#iframe
			.goto(targetURL.href, {
				referrer: originURL.origin,
				referrerPolicy: <IRequestReferrerPolicy>this.referrerPolicy
			})
			.then(() => this.dispatchEvent(new Event('load')))
			.catch((error) => WindowErrorUtility.dispatchError(this, error));

		this.#contentWindowContainer.window = isSameOrigin
			? this.#iframe.window
			: new CrossOriginBrowserWindow(this.#iframe.window, window);
	}

	/**
	 * Unloads an iframe page.
	 */
	#unloadPage(): void {
		if (this.#iframe) {
			BrowserFrameFactory.destroyFrame(this.#iframe);
			this.#iframe = null;
		}
		this.#contentWindowContainer.window = null;
		this.#loadedSrcdoc = null;
	}
}
