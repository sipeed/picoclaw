import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import Event from '../../event/Event.js';
import ErrorEvent from '../../event/events/ErrorEvent.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import Attr from '../attr/Attr.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import ResourceFetch from '../../fetch/ResourceFetch.js';
import DocumentReadyStateManager from '../document/DocumentReadyStateManager.js';
import WindowBrowserContext from '../../window/WindowBrowserContext.js';

/**
 * HTML Link Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLinkElement.
 */
export default class HTMLLinkElement extends HTMLElement {
	// Events
	public onerror: (event: ErrorEvent) => void = null;
	public onload: (event: Event) => void = null;

	// Internal properties
	public [PropertySymbol.sheet]: CSSStyleSheet = null;
	public [PropertySymbol.evaluateCSS] = true;
	public [PropertySymbol.relList]: DOMTokenList | null = null;
	#loadedStyleSheetURL: string | null = null;

	/**
	 * Returns sheet.
	 */
	public get sheet(): CSSStyleSheet {
		return this[PropertySymbol.sheet];
	}

	/**
	 * Returns rel list.
	 *
	 * @returns Rel list.
	 */
	public get relList(): DOMTokenList {
		if (!this[PropertySymbol.relList]) {
			this[PropertySymbol.relList] = new DOMTokenList(
				PropertySymbol.illegalConstructor,
				this,
				'rel'
			);
		}
		return <DOMTokenList>this[PropertySymbol.relList];
	}

	/**
	 * Returns as.
	 *
	 * @returns As.
	 */
	public get as(): string {
		return this.getAttribute('as') || '';
	}

	/**
	 * Sets crossOrigin.
	 *
	 * @param crossOrigin CrossOrigin.
	 */
	public set as(as: string) {
		this.setAttribute('as', as);
	}

	/**
	 * Returns crossOrigin.
	 *
	 * @returns CrossOrigin.
	 */
	public get crossOrigin(): string {
		return this.getAttribute('crossorigin') || '';
	}

	/**
	 * Sets crossOrigin.
	 *
	 * @param crossOrigin CrossOrigin.
	 */
	public set crossOrigin(crossOrigin: string) {
		this.setAttribute('crossorigin', crossOrigin);
	}

	/**
	 * Returns href.
	 *
	 * @returns Href.
	 */
	public get href(): string {
		if (!this.hasAttribute('href')) {
			return '';
		}

		try {
			return new URL(this.getAttribute('href'), this[PropertySymbol.ownerDocument].location.href)
				.href;
		} catch (e) {
			return this.getAttribute('href');
		}
	}

	/**
	 * Sets href.
	 *
	 * @param href Href.
	 */
	public set href(href: string) {
		this.setAttribute('href', href);
	}

	/**
	 * Returns hreflang.
	 *
	 * @returns Hreflang.
	 */
	public get hreflang(): string {
		return this.getAttribute('hreflang') || '';
	}

	/**
	 * Sets hreflang.
	 *
	 * @param hreflang Hreflang.
	 */
	public set hreflang(hreflang: string) {
		this.setAttribute('hreflang', hreflang);
	}

	/**
	 * Returns media.
	 *
	 * @returns Media.
	 */
	public get media(): string {
		return this.getAttribute('media') || '';
	}

	/**
	 * Sets media.
	 *
	 * @param media Media.
	 */
	public set media(media: string) {
		this.setAttribute('media', media);
	}

	/**
	 * Returns referrerPolicy.
	 *
	 * @returns ReferrerPolicy.
	 */
	public get referrerPolicy(): string {
		return this.getAttribute('referrerPolicy') || '';
	}

	/**
	 * Sets referrerPolicy.
	 *
	 * @param referrerPolicy ReferrerPolicy.
	 */
	public set referrerPolicy(referrerPolicy: string) {
		this.setAttribute('referrerPolicy', referrerPolicy);
	}

	/**
	 * Returns rel.
	 *
	 * @returns Rel.
	 */
	public get rel(): string {
		return this.getAttribute('rel') || '';
	}

	/**
	 * Sets rel.
	 *
	 * @param rel Rel.
	 */
	public set rel(rel: string) {
		this.setAttribute('rel', rel);
	}

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
	 * @override
	 */
	public override [PropertySymbol.connectedToDocument](): void {
		super[PropertySymbol.connectedToDocument]();
		if (this[PropertySymbol.evaluateCSS]) {
			this.#loadStyleSheet(this.getAttribute('href'), this.getAttribute('rel'));
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

		if (attribute[PropertySymbol.name] === 'rel') {
			this.#loadStyleSheet(this.getAttribute('href'), attribute[PropertySymbol.value]);
		} else if (attribute[PropertySymbol.name] === 'href') {
			this.#loadStyleSheet(attribute[PropertySymbol.value], this.getAttribute('rel'));
		}
	}

	/**
	 * Returns a URL relative to the given Location object.
	 *
	 * @param url URL.
	 * @param rel Rel.
	 */
	async #loadStyleSheet(url: string | null, rel: string | null): Promise<void> {
		const window = this[PropertySymbol.window];
		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();

		if (!browserFrame) {
			return;
		}

		const browserSettings = browserFrame.page?.context?.browser?.settings;

		if (!url || !rel || rel.toLowerCase() !== 'stylesheet' || !this[PropertySymbol.isConnected]) {
			return;
		}

		let absoluteURL: string;
		try {
			absoluteURL = new URL(url, window.location.href).href;
		} catch (error) {
			return;
		}

		if (this.#loadedStyleSheetURL === absoluteURL) {
			return;
		}

		if (browserSettings && browserSettings.disableCSSFileLoading) {
			if (browserSettings.handleDisabledFileLoadingAsSuccess) {
				this.dispatchEvent(new Event('load'));
			} else {
				WindowErrorUtility.dispatchError(
					this,
					new window.DOMException(
						`Failed to load external stylesheet "${absoluteURL}". CSS file loading is disabled.`,
						DOMExceptionNameEnum.notSupportedError
					)
				);
			}
			return;
		}

		const resourceFetch = new ResourceFetch({
			browserFrame,
			window: window
		});
		const readyStateManager = (<{ [PropertySymbol.readyStateManager]: DocumentReadyStateManager }>(
			(<unknown>window)
		))[PropertySymbol.readyStateManager];

		this.#loadedStyleSheetURL = absoluteURL;

		readyStateManager.startTask();

		let code: string | null = null;
		let error: Error | null = null;

		try {
			code = await resourceFetch.fetch(absoluteURL);
		} catch (e) {
			error = e;
		}

		readyStateManager.endTask();

		if (error) {
			WindowErrorUtility.dispatchError(this, error);
		} else {
			const styleSheet = new CSSStyleSheet();
			styleSheet.replaceSync(code);
			this[PropertySymbol.sheet] = styleSheet;

			// Computed style cache is affected by all mutations.
			const document = this[PropertySymbol.ownerDocument];
			if (document) {
				for (const item of document[PropertySymbol.affectsComputedStyleCache]) {
					item.result = null;
				}
				document[PropertySymbol.affectsComputedStyleCache] = [];
			}

			this.dispatchEvent(new Event('load'));
		}
	}
}
