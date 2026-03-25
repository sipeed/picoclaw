import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTML Hyperlink utility for HTMLAnchorElement and HTMLAreaElement.
 *
 * @see https://html.spec.whatwg.org/multipage/links.html#hyperlink
 */
export default class HTMLHyperlinkElementUtility {
	private element: HTMLElement;

	/**
	 * Constructor.
	 *
	 * @param element Element.
	 */
	constructor(element: HTMLElement) {
		this.element = element;
	}

	/**
	 * Returns the hyperlink's URL's origin.
	 *
	 * @returns Origin.
	 */
	public getOrigin(): string {
		try {
			return new URL(this.getHref()).origin;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Returns href.
	 *
	 * @returns Href.
	 */
	public getHref(): string {
		if (!this.element.hasAttribute('href')) {
			return '';
		}

		try {
			return new URL(
				this.element.getAttribute('href'),
				this.element[PropertySymbol.ownerDocument].location.href
			).href;
		} catch (e) {
			return this.element.getAttribute('href');
		}
	}

	/**
	 * Sets href.
	 *
	 * @param href Href.
	 */
	public setHref(href: string): void {
		this.element.setAttribute('href', href);
	}

	/**
	 * Returns protocol.
	 *
	 * @returns Protocol.
	 */
	public getProtocol(): string {
		try {
			return new URL(this.getHref()).protocol;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets protocol.
	 *
	 * @param protocol Protocol.
	 */
	public setProtocol(protocol: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.protocol = protocol;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns username.
	 *
	 * @returns Username.
	 */
	public getUsername(): string {
		try {
			return new URL(this.getHref()).username;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets username.
	 *
	 * @param username Username.
	 */
	public setUsername(username: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.username = username;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns password.
	 *
	 * @returns Password.
	 */
	public getPassword(): string {
		try {
			return new URL(this.getHref()).password;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets password.
	 *
	 * @param password Password.
	 */
	public setPassword(password: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.password = password;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns host.
	 *
	 * @returns Host.
	 */
	public getHost(): string {
		try {
			return new URL(this.getHref()).host;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets host.
	 *
	 * @param host Host.
	 */
	public setHost(host: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.host = host;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns hostname.
	 *
	 * @returns Hostname.
	 */
	public getHostname(): string {
		try {
			return new URL(this.getHref()).hostname;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets hostname.
	 *
	 * @param hostname Hostname.
	 */
	public setHostname(hostname: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.hostname = hostname;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns port.
	 *
	 * @returns Port.
	 */
	public getPort(): string {
		try {
			return new URL(this.getHref()).port;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets port.
	 *
	 * @param port Port.
	 */
	public setPort(port: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.port = port;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns pathname.
	 *
	 * @returns Pathname.
	 */
	public getPathname(): string {
		try {
			return new URL(this.getHref()).pathname;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets pathname.
	 *
	 * @param pathname Pathname.
	 */
	public setPathname(pathname: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.pathname = pathname;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns search.
	 *
	 * @returns Search.
	 */
	public getSearch(): string {
		try {
			return new URL(this.getHref()).search;
		} catch (e) {
			return '';
		}
	}

	/**
	 * Sets search.
	 *
	 * @param search Search.
	 */
	public setSearch(search: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.search = search;
		this.element.setAttribute('href', url.href);
	}

	/**
	 * Returns hash.
	 *
	 * @returns Hash.
	 */
	public getHash(): string {
		const href = this.element.getAttribute('href');
		if (href[0] === '#') {
			return href;
		}
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return '';
		}
		return url.hash;
	}

	/**
	 * Sets hash.
	 *
	 * @param hash Hash.
	 */
	public setHash(hash: string): void {
		let url: URL;
		try {
			url = new URL(this.getHref());
		} catch (e) {
			return;
		}
		url.hash = hash;
		this.element.setAttribute('href', url.href);
	}
}
