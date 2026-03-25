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
    get content(): string;
    /**
     * Sets content.
     *
     * @param content Content.
     */
    set content(content: string);
    /**
     * Returns httpEquiv.
     *
     * @returns HttpEquiv.
     */
    get httpEquiv(): string;
    /**
     * Sets httpEquiv.
     *
     * @param httpEquiv HttpEquiv.
     */
    set httpEquiv(httpEquiv: string);
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name(): string;
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name: string);
    /**
     * Returns scheme.
     *
     * @returns Name.
     */
    get scheme(): string;
    /**
     * Sets scheme.
     *
     * @param scheme Scheme.
     */
    set scheme(scheme: string);
}
//# sourceMappingURL=HTMLMetaElement.d.ts.map