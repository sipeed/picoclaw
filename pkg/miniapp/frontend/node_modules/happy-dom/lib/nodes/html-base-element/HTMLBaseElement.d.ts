import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML Base Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/HTML/Element/base.
 */
export default class HTMLBaseElement extends HTMLElement {
    cloneNode: (deep?: boolean) => HTMLBaseElement;
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
     * Returns target.
     *
     * @returns Target.
     */
    get target(): string;
    /**
     * Sets target.
     *
     * @param target Target.
     */
    set target(target: string);
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLBaseElement;
}
//# sourceMappingURL=HTMLBaseElement.d.ts.map