import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTML Hyperlink utility for HTMLAnchorElement and HTMLAreaElement.
 *
 * @see https://html.spec.whatwg.org/multipage/links.html#hyperlink
 */
export default class HTMLHyperlinkElementUtility {
    private element;
    /**
     * Constructor.
     *
     * @param element Element.
     */
    constructor(element: HTMLElement);
    /**
     * Returns the hyperlink's URL's origin.
     *
     * @returns Origin.
     */
    getOrigin(): string;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    getHref(): string;
    /**
     * Sets href.
     *
     * @param href Href.
     */
    setHref(href: string): void;
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    getProtocol(): string;
    /**
     * Sets protocol.
     *
     * @param protocol Protocol.
     */
    setProtocol(protocol: string): void;
    /**
     * Returns username.
     *
     * @returns Username.
     */
    getUsername(): string;
    /**
     * Sets username.
     *
     * @param username Username.
     */
    setUsername(username: string): void;
    /**
     * Returns password.
     *
     * @returns Password.
     */
    getPassword(): string;
    /**
     * Sets password.
     *
     * @param password Password.
     */
    setPassword(password: string): void;
    /**
     * Returns host.
     *
     * @returns Host.
     */
    getHost(): string;
    /**
     * Sets host.
     *
     * @param host Host.
     */
    setHost(host: string): void;
    /**
     * Returns hostname.
     *
     * @returns Hostname.
     */
    getHostname(): string;
    /**
     * Sets hostname.
     *
     * @param hostname Hostname.
     */
    setHostname(hostname: string): void;
    /**
     * Returns port.
     *
     * @returns Port.
     */
    getPort(): string;
    /**
     * Sets port.
     *
     * @param port Port.
     */
    setPort(port: string): void;
    /**
     * Returns pathname.
     *
     * @returns Pathname.
     */
    getPathname(): string;
    /**
     * Sets pathname.
     *
     * @param pathname Pathname.
     */
    setPathname(pathname: string): void;
    /**
     * Returns search.
     *
     * @returns Search.
     */
    getSearch(): string;
    /**
     * Sets search.
     *
     * @param search Search.
     */
    setSearch(search: string): void;
    /**
     * Returns hash.
     *
     * @returns Hash.
     */
    getHash(): string;
    /**
     * Sets hash.
     *
     * @param hash Hash.
     */
    setHash(hash: string): void;
}
//# sourceMappingURL=HTMLHyperlinkElementUtility.d.ts.map