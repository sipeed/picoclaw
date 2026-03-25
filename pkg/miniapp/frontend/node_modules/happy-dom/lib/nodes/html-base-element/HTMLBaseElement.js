import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML Base Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/HTML/Element/base.
 */
export default class HTMLBaseElement extends HTMLElement {
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this.hasAttribute('href')) {
            return this[PropertySymbol.ownerDocument].location.href;
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
     * Returns target.
     *
     * @returns Target.
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
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=HTMLBaseElement.js.map