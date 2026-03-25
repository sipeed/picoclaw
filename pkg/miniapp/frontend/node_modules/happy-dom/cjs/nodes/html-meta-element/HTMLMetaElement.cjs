"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
/**
 * HTML Meta Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLMetaElement.
 */
class HTMLMetaElement extends HTMLElement_js_1.default {
    /**
     * Returns content.
     *
     * @returns Content.
     */
    get content() {
        return this.getAttribute('content') || '';
    }
    /**
     * Sets content.
     *
     * @param content Content.
     */
    set content(content) {
        this.setAttribute('content', content);
    }
    /**
     * Returns httpEquiv.
     *
     * @returns HttpEquiv.
     */
    get httpEquiv() {
        return this.getAttribute('http-equiv') || '';
    }
    /**
     * Sets httpEquiv.
     *
     * @param httpEquiv HttpEquiv.
     */
    set httpEquiv(httpEquiv) {
        this.setAttribute('http-equiv', httpEquiv);
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this.getAttribute('name') || '';
    }
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name) {
        this.setAttribute('name', name);
    }
    /**
     * Returns scheme.
     *
     * @returns Name.
     */
    get scheme() {
        return this.getAttribute('scheme') || '';
    }
    /**
     * Sets scheme.
     *
     * @param scheme Scheme.
     */
    set scheme(scheme) {
        this.setAttribute('scheme', scheme);
    }
}
exports.default = HTMLMetaElement;
//# sourceMappingURL=HTMLMetaElement.cjs.map