import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import Event from '../../event/Event.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import ResourceFetch from '../../fetch/ResourceFetch.js';
import WindowBrowserContext from '../../window/WindowBrowserContext.js';
/**
 * HTML Link Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLinkElement.
 */
export default class HTMLLinkElement extends HTMLElement {
    // Events
    onerror = null;
    onload = null;
    // Internal properties
    [PropertySymbol.sheet] = null;
    [PropertySymbol.evaluateCSS] = true;
    [PropertySymbol.relList] = null;
    #loadedStyleSheetURL = null;
    /**
     * Returns sheet.
     */
    get sheet() {
        return this[PropertySymbol.sheet];
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
     * Returns as.
     *
     * @returns As.
     */
    get as() {
        return this.getAttribute('as') || '';
    }
    /**
     * Sets crossOrigin.
     *
     * @param crossOrigin CrossOrigin.
     */
    set as(as) {
        this.setAttribute('as', as);
    }
    /**
     * Returns crossOrigin.
     *
     * @returns CrossOrigin.
     */
    get crossOrigin() {
        return this.getAttribute('crossorigin') || '';
    }
    /**
     * Sets crossOrigin.
     *
     * @param crossOrigin CrossOrigin.
     */
    set crossOrigin(crossOrigin) {
        this.setAttribute('crossorigin', crossOrigin);
    }
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this.hasAttribute('href')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('href'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('href');
        }
    }
    /**
     * Sets href.
     *
     * @param href Href.
     */
    set href(href) {
        this.setAttribute('href', href);
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
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        return this.getAttribute('media') || '';
    }
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media) {
        this.setAttribute('media', media);
    }
    /**
     * Returns referrerPolicy.
     *
     * @returns ReferrerPolicy.
     */
    get referrerPolicy() {
        return this.getAttribute('referrerPolicy') || '';
    }
    /**
     * Sets referrerPolicy.
     *
     * @param referrerPolicy ReferrerPolicy.
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
    [PropertySymbol.connectedToDocument]() {
        super[PropertySymbol.connectedToDocument]();
        if (this[PropertySymbol.evaluateCSS]) {
            this.#loadStyleSheet(this.getAttribute('href'), this.getAttribute('rel'));
        }
    }
    /**
     * @override
     */
    [PropertySymbol.onSetAttribute](attribute, replacedAttribute) {
        super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
        if (attribute[PropertySymbol.name] === 'rel') {
            this.#loadStyleSheet(this.getAttribute('href'), attribute[PropertySymbol.value]);
        }
        else if (attribute[PropertySymbol.name] === 'href') {
            this.#loadStyleSheet(attribute[PropertySymbol.value], this.getAttribute('rel'));
        }
    }
    /**
     * Returns a URL relative to the given Location object.
     *
     * @param url URL.
     * @param rel Rel.
     */
    async #loadStyleSheet(url, rel) {
        const window = this[PropertySymbol.window];
        const browserFrame = new WindowBrowserContext(window).getBrowserFrame();
        if (!browserFrame) {
            return;
        }
        const browserSettings = browserFrame.page?.context?.browser?.settings;
        if (!url || !rel || rel.toLowerCase() !== 'stylesheet' || !this[PropertySymbol.isConnected]) {
            return;
        }
        let absoluteURL;
        try {
            absoluteURL = new URL(url, window.location.href).href;
        }
        catch (error) {
            return;
        }
        if (this.#loadedStyleSheetURL === absoluteURL) {
            return;
        }
        if (browserSettings && browserSettings.disableCSSFileLoading) {
            if (browserSettings.handleDisabledFileLoadingAsSuccess) {
                this.dispatchEvent(new Event('load'));
            }
            else {
                WindowErrorUtility.dispatchError(this, new window.DOMException(`Failed to load external stylesheet "${absoluteURL}". CSS file loading is disabled.`, DOMExceptionNameEnum.notSupportedError));
            }
            return;
        }
        const resourceFetch = new ResourceFetch({
            browserFrame,
            window: window
        });
        const readyStateManager = window[PropertySymbol.readyStateManager];
        this.#loadedStyleSheetURL = absoluteURL;
        readyStateManager.startTask();
        let code = null;
        let error = null;
        try {
            code = await resourceFetch.fetch(absoluteURL);
        }
        catch (e) {
            error = e;
        }
        readyStateManager.endTask();
        if (error) {
            WindowErrorUtility.dispatchError(this, error);
        }
        else {
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
//# sourceMappingURL=HTMLLinkElement.js.map