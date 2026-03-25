"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * HTML Hyperlink utility for HTMLAnchorElement and HTMLAreaElement.
 *
 * @see https://html.spec.whatwg.org/multipage/links.html#hyperlink
 */
class HTMLHyperlinkElementUtility {
    element;
    /**
     * Constructor.
     *
     * @param element Element.
     */
    constructor(element) {
        this.element = element;
    }
    /**
     * Returns the hyperlink's URL's origin.
     *
     * @returns Origin.
     */
    getOrigin() {
        try {
            return new URL(this.getHref()).origin;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Returns href.
     *
     * @returns Href.
     */
    getHref() {
        if (!this.element.hasAttribute('href')) {
            return '';
        }
        try {
            return new URL(this.element.getAttribute('href'), this.element[PropertySymbol.ownerDocument].location.href).href;
        }
        catch (e) {
            return this.element.getAttribute('href');
        }
    }
    /**
     * Sets href.
     *
     * @param href Href.
     */
    setHref(href) {
        this.element.setAttribute('href', href);
    }
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    getProtocol() {
        try {
            return new URL(this.getHref()).protocol;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets protocol.
     *
     * @param protocol Protocol.
     */
    setProtocol(protocol) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getUsername() {
        try {
            return new URL(this.getHref()).username;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets username.
     *
     * @param username Username.
     */
    setUsername(username) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getPassword() {
        try {
            return new URL(this.getHref()).password;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets password.
     *
     * @param password Password.
     */
    setPassword(password) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getHost() {
        try {
            return new URL(this.getHref()).host;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets host.
     *
     * @param host Host.
     */
    setHost(host) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getHostname() {
        try {
            return new URL(this.getHref()).hostname;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets hostname.
     *
     * @param hostname Hostname.
     */
    setHostname(hostname) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getPort() {
        try {
            return new URL(this.getHref()).port;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets port.
     *
     * @param port Port.
     */
    setPort(port) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getPathname() {
        try {
            return new URL(this.getHref()).pathname;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets pathname.
     *
     * @param pathname Pathname.
     */
    setPathname(pathname) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getSearch() {
        try {
            return new URL(this.getHref()).search;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets search.
     *
     * @param search Search.
     */
    setSearch(search) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
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
    getHash() {
        const href = this.element.getAttribute('href');
        if (href[0] === '#') {
            return href;
        }
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
            return '';
        }
        return url.hash;
    }
    /**
     * Sets hash.
     *
     * @param hash Hash.
     */
    setHash(hash) {
        let url;
        try {
            url = new URL(this.getHref());
        }
        catch (e) {
            return;
        }
        url.hash = hash;
        this.element.setAttribute('href', url.href);
    }
}
exports.default = HTMLHyperlinkElementUtility;
//# sourceMappingURL=HTMLHyperlinkElementUtility.cjs.map