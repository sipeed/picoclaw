import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTML Meta Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLMetaElement.
 */
export default class HTMLMetaElement extends HTMLElement {
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
//# sourceMappingURL=HTMLMetaElement.js.map