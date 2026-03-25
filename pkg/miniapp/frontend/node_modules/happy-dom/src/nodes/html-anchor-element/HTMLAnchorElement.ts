import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import Event from '../../event/Event.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import HTMLHyperlinkElementUtility from '../html-hyperlink-element/HTMLHyperlinkElementUtility.js';
import IHTMLHyperlinkElement from '../html-hyperlink-element/IHTMLHyperlinkElement.js';
import MouseEvent from '../../event/events/MouseEvent.js';

/**
 * HTML Anchor Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLAnchorElement.
 */
export default class HTMLAnchorElement extends HTMLElement implements IHTMLHyperlinkElement {
	public [PropertySymbol.relList]: DOMTokenList | null = null;
	#htmlHyperlinkElementUtility = new HTMLHyperlinkElementUtility(this);

	/**
	 * Returns download.
	 *
	 * @returns download.
	 */
	public get download(): string {
		return this.getAttribute('download') || '';
	}

	/**
	 * Sets download.
	 *
	 * @param download Download.
	 */
	public set download(download: string) {
		this.setAttribute('download', download);
	}

	/**
	 * Returns hash.
	 *
	 * @returns Hash.
	 */
	public get hash(): string {
		return this.#htmlHyperlinkElementUtility.getHash();
	}

	/**
	 * Sets hash.
	 *
	 * @param hash Hash.
	 */
	public set hash(hash: string) {
		this.#htmlHyperlinkElementUtility.setHash(hash);
	}

	/**
	 * Returns href.
	 *
	 * @returns Href.
	 */
	public get href(): string {
		return this.#htmlHyperlinkElementUtility.getHref();
	}

	/**
	 * Sets href.
	 *
	 * @param href Href.
	 */
	public set href(href: string) {
		this.#htmlHyperlinkElementUtility.setHref(href);
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
	 * Returns the hyperlink's URL's origin.
	 *
	 * @returns Origin.
	 */
	public get origin(): string {
		return this.#htmlHyperlinkElementUtility.getOrigin();
	}

	/**
	 * Returns ping.
	 *
	 * @returns Ping.
	 */
	public get ping(): string {
		return this.getAttribute('ping') || '';
	}

	/**
	 * Sets ping.
	 *
	 * @param ping Ping.
	 */
	public set ping(ping: string) {
		this.setAttribute('ping', ping);
	}

	/**
	 * Returns protocol.
	 *
	 * @returns Protocol.
	 */
	public get protocol(): string {
		return this.#htmlHyperlinkElementUtility.getProtocol();
	}

	/**
	 * Sets protocol.
	 *
	 * @param protocol Protocol.
	 */
	public set protocol(protocol: string) {
		this.#htmlHyperlinkElementUtility.setProtocol(protocol);
	}

	/**
	 * Returns username.
	 *
	 * @returns Username.
	 */
	public get username(): string {
		return this.#htmlHyperlinkElementUtility.getUsername();
	}

	/**
	 * Sets username.
	 *
	 * @param username Username.
	 */
	public set username(username: string) {
		this.#htmlHyperlinkElementUtility.setUsername(username);
	}

	/**
	 * Returns password.
	 *
	 * @returns Password.
	 */
	public get password(): string {
		return this.#htmlHyperlinkElementUtility.getPassword();
	}

	/**
	 * Sets password.
	 *
	 * @param password Password.
	 */
	public set password(password: string) {
		this.#htmlHyperlinkElementUtility.setPassword(password);
	}

	/**
	 * Returns pathname.
	 *
	 * @returns Pathname.
	 */
	public get pathname(): string {
		return this.#htmlHyperlinkElementUtility.getPathname();
	}

	/**
	 * Sets pathname.
	 *
	 * @param pathname Pathname.
	 */
	public set pathname(pathname: string) {
		this.#htmlHyperlinkElementUtility.setPathname(pathname);
	}

	/**
	 * Returns port.
	 *
	 * @returns Port.
	 */
	public get port(): string {
		return this.#htmlHyperlinkElementUtility.getPort();
	}

	/**
	 * Sets port.
	 *
	 * @param port Port.
	 */
	public set port(port: string) {
		this.#htmlHyperlinkElementUtility.setPort(port);
	}

	/**
	 * Returns host.
	 *
	 * @returns Host.
	 */
	public get host(): string {
		return this.#htmlHyperlinkElementUtility.getHost();
	}

	/**
	 * Sets host.
	 *
	 * @param host Host.
	 */
	public set host(host: string) {
		this.#htmlHyperlinkElementUtility.setHost(host);
	}

	/**
	 * Returns hostname.
	 *
	 * @returns Hostname.
	 */
	public get hostname(): string {
		return this.#htmlHyperlinkElementUtility.getHostname();
	}

	/**
	 * Sets hostname.
	 *
	 * @param hostname Hostname.
	 */
	public set hostname(hostname: string) {
		this.#htmlHyperlinkElementUtility.setHostname(hostname);
	}

	/**
	 * Returns referrerPolicy.
	 *
	 * @returns Referrer Policy.
	 */
	public get referrerPolicy(): string {
		return this.getAttribute('referrerPolicy') || '';
	}

	/**
	 * Sets referrerPolicy.
	 *
	 * @param referrerPolicy Referrer Policy.
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
	 * Returns search.
	 *
	 * @returns Search.
	 */
	public get search(): string {
		return this.#htmlHyperlinkElementUtility.getSearch();
	}

	/**
	 * Sets search.
	 *
	 * @param search Search.
	 */
	public set search(search: string) {
		this.#htmlHyperlinkElementUtility.setSearch(search);
	}

	/**
	 * Returns target.
	 *
	 * @returns target.
	 */
	public get target(): string {
		return this.getAttribute('target') || '';
	}

	/**
	 * Sets target.
	 *
	 * @param target Target.
	 */
	public set target(target: string) {
		this.setAttribute('target', target);
	}

	/**
	 * Returns text.
	 *
	 * @returns text.
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
	public override toString(): string {
		return this.href;
	}

	/**
	 * @override
	 */
	public override dispatchEvent(event: Event): boolean {
		const returnValue = super.dispatchEvent(event);

		if (
			!event[PropertySymbol.defaultPrevented] &&
			event.type === 'click' &&
			event instanceof MouseEvent &&
			event.eventPhase === EventPhaseEnum.none
		) {
			const href = this.href;

			if (href) {
				const features = [];

				if (this.relList.contains('noreferrer')) {
					features.push('noreferrer');
				}

				if (this.relList.contains('noopener')) {
					features.push('noopener');
				}

				this[PropertySymbol.window].open(href, this.target || '_self', features.join(','));

				if (this[PropertySymbol.window].closed) {
					event.stopImmediatePropagation();
				}
			}
		}

		return returnValue;
	}
}
