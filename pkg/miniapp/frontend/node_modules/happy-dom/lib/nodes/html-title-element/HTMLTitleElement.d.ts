import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTMLTitleElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTitleElement
 */
export default class HTMLTitleElement extends HTMLElement {
    /**
     * Returns text.
     *
     * @returns Text.
     */
    get text(): string;
    /**
     * Sets text.
     *
     * @param text Text.
     */
    set text(text: string);
    /**
     * @override
     */
    get innerHTML(): string;
    /**
     * @override
     */
    set innerHTML(html: string);
}
//# sourceMappingURL=HTMLTitleElement.d.ts.map