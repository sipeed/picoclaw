import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTMLModElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLModElement
 */
export default class HTMLModElement extends HTMLElement {
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
    /**
     * Returns date time.
     *
     * @returns Date time.
     */
    get dateTime() {
        return this.getAttribute('datetime') || '';
    }
    /**
     * Sets date time.
     *
     * @param dateTime Date time.
     */
    set dateTime(dateTime) {
        this.setAttribute('datetime', dateTime);
    }
}
//# sourceMappingURL=HTMLModElement.js.map