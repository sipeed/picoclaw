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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CSSStyleSheet_js_1 = __importDefault(require("../../css/CSSStyleSheet.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const DOMTokenList_js_1 = __importDefault(require("../../dom/DOMTokenList.cjs"));
const WindowErrorUtility_js_1 = __importDefault(require("../../window/WindowErrorUtility.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const ResourceFetch_js_1 = __importDefault(require("../../fetch/ResourceFetch.cjs"));
const WindowBrowserContext_js_1 = __importDefault(require("../../window/WindowBrowserContext.cjs"));
/**
 * HTML Link Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLinkElement.
 */
class HTMLLinkElement extends HTMLElement_js_1.default {
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
            this[PropertySymbol.relList] = new DOMTokenList_js_1.default(PropertySymbol.illegalConstructor, this, 'rel');
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
        const browserFrame = new WindowBrowserContext_js_1.default(window).getBrowserFrame();
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
                this.dispatchEvent(new Event_js_1.default('load'));
            }
            else {
                WindowErrorUtility_js_1.default.dispatchError(this, new window.DOMException(`Failed to load external stylesheet "${absoluteURL}". CSS file loading is disabled.`, DOMExceptionNameEnum_js_1.default.notSupportedError));
            }
            return;
        }
        const resourceFetch = new ResourceFetch_js_1.default({
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
            WindowErrorUtility_js_1.default.dispatchError(this, error);
        }
        else {
            const styleSheet = new CSSStyleSheet_js_1.default();
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
            this.dispatchEvent(new Event_js_1.default('load'));
        }
    }
}
exports.default = HTMLLinkElement;
//# sourceMappingURL=HTMLLinkElement.cjs.map