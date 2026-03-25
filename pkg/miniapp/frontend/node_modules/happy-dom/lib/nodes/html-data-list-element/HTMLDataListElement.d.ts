import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLCollection from '../element/HTMLCollection.js';
import HTMLOptionElement from '../html-option-element/HTMLOptionElement.js';
/**
 * HTMLDataListElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDataListElement
 */
export default class HTMLDataListElement extends HTMLElement {
    [PropertySymbol.options]: HTMLCollection<HTMLOptionElement> | null;
    /**
     * Returns options.
     *
     * @returns Options.
     */
    get options(): HTMLCollection<HTMLOptionElement>;
}
//# sourceMappingURL=HTMLDataListElement.d.ts.map