import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTMLQuoteElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLQuoteElement
 */
export default class HTMLQuoteElement extends HTMLElement {
    /**
     * Returns source.
     *
     * @returns Source.
     */
    get cite() {
        if (!this.hasAttribute('cite')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('cite'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('cite');
        }
    }
    /**
     * Sets source.
     *
     * @param cite Source.
     */
    set cite(cite) {
        this.setAttribute('cite', cite);
    }
}
//# sourceMappingURL=HTMLQuoteElement.js.map