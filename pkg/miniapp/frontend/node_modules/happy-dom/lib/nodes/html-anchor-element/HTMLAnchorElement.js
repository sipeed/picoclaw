import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import HTMLHyperlinkElementUtility from '../html-hyperlink-element/HTMLHyperlinkElementUtility.js';
import MouseEvent from '../../event/events/MouseEvent.js';
/**
 * HTML Anchor Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLAnchorElement.
 */
export default class HTMLAnchorElement extends HTMLElement {
    [PropertySymbol.relList] = null;
    #htmlHyperlinkElementUtility = new HTMLHyperlinkElementUtility(this);
    /**
     * Returns download.
     *
     * @returns download.
     */
    get download() {
        return this.getAttribute('download') || '';
    }
    /**
     * Sets download.
     *
     * @param download Download.
     */
    set download(download) {
        this.setAttribute('download', download);
    }
    /**
     * Returns hash.
     *
     * @returns Hash.
     */
    get hash() {
        return this.#htmlHyperlinkElementUtility.getHash();
    }
    /**
     * Sets hash.
     *
     * @param hash Hash.
     */
    set hash(hash) {
        this.#htmlHyperlinkElementUtility.setHash(hash);
    }
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        return this.#htmlHyperlinkElementUtility.getHref();
    }
    /**
     * Sets href.
     *
     * @param href Href.
     */
    set href(href) {
        this.#htmlHyperlinkElementUtility.setHref(href);
    }
    /**
     * Returns hreflang.
     *
     * @returns Hreflang.
     */
    get hreflang() {
        return this.getAttribute('hreflang') || '';
    }
    /**
     * Sets hreflang.
     *
     * @param hreflang Hreflang.
     */
    set hreflang(hreflang) {
        this.setAttribute('hreflang', hreflang);
    }
    /**
     * Returns the hyperlink's URL's origin.
     *
     * @returns Origin.
     */
    get origin() {
        return this.#htmlHyperlinkElementUtility.getOrigin();
    }
    /**
     * Returns ping.
     *
     * @returns Ping.
     */
    get ping() {
        return this.getAttribute('ping') || '';
    }
    /**
     * Sets ping.
     *
     * @param ping Ping.
     */
    set ping(ping) {
        this.setAttribute('ping', ping);
    }
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    get protocol() {
        return this.#htmlHyperlinkElementUtility.getProtocol();
    }
    /**
     * Sets protocol.
     *
     * @param protocol Protocol.
     */
    set protocol(protocol) {
        this.#htmlHyperlinkElementUtility.setProtocol(protocol);
    }
    /**
     * Returns username.
     *
     * @returns Username.
     */
    get username() {
        return this.#htmlHyperlinkElementUtility.getUsername();
    }
    /**
     * Sets username.
     *
     * @param username Username.
     */
    set username(username) {
        this.#htmlHyperlinkElementUtility.setUsername(username);
    }
    /**
     * Returns password.
     *
     * @returns Password.
     */
    get password() {
        return this.#htmlHyperlinkElementUtility.getPassword();
    }
    /**
     * Sets password.
     *
     * @param password Password.
     */
    set password(password) {
        this.#htmlHyperlinkElementUtility.setPassword(password);
    }
    /**
     * Returns pathname.
     *
     * @returns Pathname.
     */
    get pathname() {
        return this.#htmlHyperlinkElementUtility.getPathname();
    }
    /**
     * Sets pathname.
     *
     * @param pathname Pathname.
     */
    set pathname(pathname) {
        this.#htmlHyperlinkElementUtility.setPathname(pathname);
    }
    /**
     * Returns port.
     *
     * @returns Port.
     */
    get port() {
        return this.#htmlHyperlinkElementUtility.getPort();
    }
    /**
     * Sets port.
     *
     * @param port Port.
     */
    set port(port) {
        this.#htmlHyperlinkElementUtility.setPort(port);
    }
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host() {
        return this.#htmlHyperlinkElementUtility.getHost();
    }
    /**
     * Sets host.
     *
     * @param host Host.
     */
    set host(host) {
        this.#htmlHyperlinkElementUtility.setHost(host);
    }
    /**
     * Returns hostname.
     *
     * @returns Hostname.
     */
    get hostname() {
        return this.#htmlHyperlinkElementUtility.getHostname();
    }
    /**
     * Sets hostname.
     *
     * @param hostname Hostname.
     */
    set hostname(hostname) {
        this.#htmlHyperlinkElementUtility.setHostname(hostname);
    }
    /**
     * Returns referrerPolicy.
     *
     * @returns Referrer Policy.
     */
    get referrerPolicy() {
        return this.getAttribute('referrerPolicy') || '';
    }
    /**
     * Sets referrerPolicy.
     *
     * @param referrerPolicy Referrer Policy.
     */
    set referrerPolicy(referrerPolicy) {
        this.setAttribute('referrerPolicy', referrerPolicy);
    }
    /**
     * Returns rel.
     *
     * @returns Rel.
     */
    get rel() {
        return this.getAttribute('rel') || '';
    }
    /**
     * Sets rel.
     *
     * @param rel Rel.
     */
    set rel(rel) {
        this.setAttribute('rel', rel);
    }
    /**
     * Returns rel list.
     *
     * @returns Rel list.
     */
    get relList() {
        if (!this[PropertySymbol.relList]) {
            this[PropertySymbol.relList] = new DOMTokenList(PropertySymbol.illegalConstructor, this, 'rel');
        }
        return this[PropertySymbol.relList];
    }
    /**
     * Returns search.
     *
     * @returns Search.
     */
    get search() {
        return this.#htmlHyperlinkElementUtility.getSearch();
    }
    /**
     * Sets search.
     *
     * @param search Search.
     */
    set search(search) {
        this.#htmlHyperlinkElementUtility.setSearch(search);
    }
    /**
     * Returns target.
     *
     * @returns target.
     */
    get target() {
        return this.getAttribute('target') || '';
    }
    /**
     * Sets target.
     *
     * @param target Target.
     */
    set target(target) {
        this.setAttribute('target', target);
    }
    /**
     * Returns text.
     *
     * @returns text.
     */
    get text() {
        return this.textContent;
    }
    /**
     * Sets text.
     *
     * @param text Text.
     */
    set text(text) {
        this.textContent = text;
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
    /**
     * @override
     */
    toString() {
        return this.href;
    }
    /**
     * @override
     */
    dispatchEvent(event) {
        const returnValue = super.dispatchEvent(event);
        if (!event[PropertySymbol.defaultPrevented] &&
            event.type === 'click' &&
            event instanceof MouseEvent &&
            event.eventPhase === EventPhaseEnum.none) {
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
//# sourceMappingURL=HTMLAnchorElement.js.map