import HTMLElement from '../html-element/HTMLElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import DOMTokenList from '../../dom/DOMTokenList.cjs';
import Event from '../../event/Event.cjs';
import IHTMLHyperlinkElement from '../html-hyperlink-element/IHTMLHyperlinkElement.cjs';
/**
 * HTML Anchor Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLAnchorElement.
 */
export default class HTMLAnchorElement extends HTMLElement implements IHTMLHyperlinkElement {
    #private;
    [PropertySymbol.relList]: DOMTokenList | null;
    /**
     * Returns download.
     *
     * @returns download.
     */
    get download(): string;
    /**
     * Sets download.
     *
     * @param download Download.
     */
    set download(download: string);
    /**
     * Returns hash.
     *
     * @returns Hash.
     */
    get hash(): string;
    /**
     * Sets hash.
     *
     * @param hash Hash.
     */
    set hash(hash: string);
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): string;
    /**
     * Sets href.
     *
     * @param href Href.
     */
    set href(href: string);
    /**
     * Returns hreflang.
     *
     * @returns Hreflang.
     */
    get hreflang(): string;
    /**
     * Sets hreflang.
     *
     * @param hreflang Hreflang.
     */
    set hreflang(hreflang: string);
    /**
     * Returns the hyperlink's URL's origin.
     *
     * @returns Origin.
     */
    get origin(): string;
    /**
     * Returns ping.
     *
     * @returns Ping.
     */
    get ping(): string;
    /**
     * Sets ping.
     *
     * @param ping Ping.
     */
    set ping(ping: string);
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    get protocol(): string;
    /**
     * Sets protocol.
     *
     * @param protocol Protocol.
     */
    set protocol(protocol: string);
    /**
     * Returns username.
     *
     * @returns Username.
     */
    get username(): string;
    /**
     * Sets username.
     *
     * @param username Username.
     */
    set username(username: string);
    /**
     * Returns password.
     *
     * @returns Password.
     */
    get password(): string;
    /**
     * Sets password.
     *
     * @param password Password.
     */
    set password(password: string);
    /**
     * Returns pathname.
     *
     * @returns Pathname.
     */
    get pathname(): string;
    /**
     * Sets pathname.
     *
     * @param pathname Pathname.
     */
    set pathname(pathname: string);
    /**
     * Returns port.
     *
     * @returns Port.
     */
    get port(): string;
    /**
     * Sets port.
     *
     * @param port Port.
     */
    set port(port: string);
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host(): string;
    /**
     * Sets host.
     *
     * @param host Host.
     */
    set host(host: string);
    /**
     * Returns hostname.
     *
     * @returns Hostname.
     */
    get hostname(): string;
    /**
     * Sets hostname.
     *
     * @param hostname Hostname.
     */
    set hostname(hostname: string);
    /**
     * Returns referrerPolicy.
     *
     * @returns Referrer Policy.
     */
    get referrerPolicy(): string;
    /**
     * Sets referrerPolicy.
     *
     * @param referrerPolicy Referrer Policy.
     */
    set referrerPolicy(referrerPolicy: string);
    /**
     * Returns rel.
     *
     * @returns Rel.
     */
    get rel(): string;
    /**
     * Sets rel.
     *
     * @param rel Rel.
     */
    set rel(rel: string);
    /**
     * Returns rel list.
     *
     * @returns Rel list.
     */
    get relList(): DOMTokenList;
    /**
     * Returns search.
     *
     * @returns Search.
     */
    get search(): string;
    /**
     * Sets search.
     *
     * @param search Search.
     */
    set search(search: string);
    /**
     * Returns target.
     *
     * @returns target.
     */
    get target(): string;
    /**
     * Sets target.
     *
     * @param target Target.
     */
    set target(target: string);
    /**
     * Returns text.
     *
     * @returns text.
     */
    get text(): string;
    /**
     * Sets text.
     *
     * @param text Text.
     */
    set text(text: string);
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): string;
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type: string);
    /**
     * @override
     */
    toString(): string;
    /**
     * @override
     */
    dispatchEvent(event: Event): boolean;
}
//# sourceMappingURL=HTMLAnchorElement.d.ts.map